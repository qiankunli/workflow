package operators

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sutilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/constants"
	"github.com/qiankunli/workflow/pkg/utils/kube"
	"github.com/qiankunli/workflow/pkg/utils/mutex"

	"github.com/qiankunli/workflow/pkg/controller/manager"
)

type workflowReconciler struct {
	client        client.Client
	controllerCtx *manager.ControllerContext
	log           logr.Logger
	recorder      record.EventRecorder
	WorkflowMutex mutex.GroupMutex
}

// RegisterWorkflowReconciler ...
func RegisterWorkflowReconciler(mgr ctrl.Manager, controllerCtx *manager.ControllerContext) error {
	const name = "workflow-controller"

	r := &workflowReconciler{
		client:        mgr.GetClient(),
		controllerCtx: controllerCtx,
		log:           ctrl.LoggerFrom(context.Background()).WithName(name),
		recorder:      mgr.GetEventRecorderFor(name),
		WorkflowMutex: controllerCtx.WorkflowMutex,
	}

	_, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			CacheSyncTimeout:        controllerCtx.Config.ControllerConfig.SyncTimeout.Duration,
			MaxConcurrentReconciles: controllerCtx.Config.ControllerConfig.Concurrency,
		}).
		For(&v1alpha1.Workflow{}).
		Watches(&source.Kind{Type: &v1alpha1.Step{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &v1alpha1.Workflow{},
		}).
		Named(name).
		Build(r)

	if err != nil {
		return fmt.Errorf("failed to set up with manager: %w", err)
	}
	klog.InfoS("succeeded to set up with manager")
	return nil
}

// Reconcile ...
func (r *workflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, reterr error) {
	log := r.log.WithValues("name", req.Name)
	defer func() {
		if p := recover(); p != nil {
			log.Error(errors.Errorf("panic error: %+v", p), "Panic error")
			debug.PrintStack()
		}
	}()

	// Only one reconcile routine can deal for per cr
	lockKey := req.Namespace + req.Name
	if !r.WorkflowMutex.Lock(lockKey) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	defer r.WorkflowMutex.Unlock(lockKey)

	ctx = ctrl.LoggerInto(ctx, log)
	log.V(4).Info("workflow start reconcile")

	// Fetch the workflow
	workflow := &v1alpha1.Workflow{}
	if err := r.client.Get(ctx, req.NamespacedName, workflow); err != nil {
		if k8sapierrors.IsNotFound(err) {
			klog.InfoS("workflow has been deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get workflow")
		return ctrl.Result{}, err
	}
	log.V(4).Info("step start reconcile", "phase", workflow.Status.Phase)

	// Initialize the patch helper, defer patch the workflow
	patchHelper, err := kube.NewHelper(workflow, r.client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := patchHelper.Patch(ctx, workflow); err != nil {
			reterr = k8sutilerrors.NewAggregate([]error{reterr, err})
		}
	}()

	steps, err := r.GetStepsForWorkflow(workflow)
	if err != nil {
		log.Error(err, "find steps for workflow error")
		return ctrl.Result{}, err
	}
	// 根据step 状态更新下workflow 状态以便决定下一步逻辑
	r.aggregateStepStatus(ctx, workflow, steps)
	if !workflow.DeletionTimestamp.IsZero() {
		log.V(4).Info("workflow deletionTimestamp is not zero", "phase", workflow.Status.Phase)
		if seeAsRollBackedWorkflow(workflow) {
			if err = r.onDeleted(ctx, workflow); err != nil {
				return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
			}
			controllerutil.RemoveFinalizer(workflow, constants.FinalizersWorkflow)
			r.WorkflowMutex.DelMutex(lockKey)
			return ctrl.Result{}, nil
		}
		// 回滚失败
		if workflow.Status.Phase == v1alpha1.WorkflowFailed {
			// 回滚失败需要要多次告知vector，用户可能会多次点释放
			if err = r.onRollback(ctx, workflow); err != nil {
				return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
			}
			// 如果发现运行中、已成功的step，则触发其回滚
			if workflow.Status.StepPhases[v1alpha1.StepRunning]+workflow.Status.StepPhases[v1alpha1.StepSuccess] > 0 {
				r.reconcileRollingBack(ctx, workflow, steps)
				return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
			}
			// failed之后，建议人工介入处理，无论wf 还是step 都不会对failed 状态再施加操作，否则逻辑太复杂了
			return ctrl.Result{}, nil
		}
		// 回滚中
		r.reconcileRollingBack(ctx, workflow, steps)
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}
	if !controllerutil.ContainsFinalizer(workflow, constants.FinalizersWorkflow) {
		controllerutil.AddFinalizer(workflow, constants.FinalizersWorkflow)
	}
	if workflow.Status.Phase == v1alpha1.WorkflowRunning {
		r.reconcileCreating(ctx, workflow, steps)
		r.reconcileRunning(ctx, workflow, steps)
		if err = r.onStart(ctx, workflow); err != nil {
			return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
		}
		// 进行态要一会儿再进来看下
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}
	if seeAsRollingBackWorkflow(workflow) {
		r.reconcileRollingBack(ctx, workflow, steps)
		// 进行态要一会儿再进来看下
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}
	if workflow.Status.Phase == v1alpha1.WorkflowFailed || workflow.Status.Phase == v1alpha1.WorkflowRollBacked {
		// 上报失败状态，如果上报失败，则持续上报
		if err = r.onRollback(ctx, workflow); err != nil {
			return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
		}
		// 如果发现运行中、已成功的step，则触发其回滚
		if workflow.Status.StepPhases[v1alpha1.StepRunning]+workflow.Status.StepPhases[v1alpha1.StepSuccess] > 0 {
			r.reconcileRollingBack(ctx, workflow, steps)
			// 进行态要一会儿再进来看下
			return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
		}
	}
	// 静止态，不用触发下次reconcile了
	if workflow.Status.Phase == v1alpha1.WorkflowSuccess {
		if err = r.onSuccess(ctx, workflow); err != nil {
			// 触发callback失败，要一会儿再进来看下
			return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (r *workflowReconciler) aggregateStepStatus(ctx context.Context, workflow *v1alpha1.Workflow, steps []v1alpha1.Step) {
	log := r.log.WithValues("name", workflow.Name)
	currentPhase := workflow.Status.Phase
	if len(steps) == 0 {
		log.V(4).Info("can not find steps for workflow")
		return
	}
	count := map[v1alpha1.StepPhase]int{}
	runErrors := make([]string, 0)
	rollbackErrors := make([]string, 0)
	syncErrors := make([]string, 0)
	stepAttributes := map[string]string{}
	for _, step := range steps {
		count[step.Status.Phase]++
		if len(step.Status.RunError) > 0 {
			runErrors = append(runErrors, fmt.Sprintf("%s:%s", step.Spec.Type, step.Status.RunError))
		}
		if len(step.Status.RollbackError) > 0 {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("%s:%s", step.Spec.Type, step.Status.RollbackError))
		}
		if len(step.Status.SyncError) > 0 {
			syncErrors = append(syncErrors, fmt.Sprintf("%s:%s", step.Spec.Type, step.Status.SyncError))
		}
		for k, v := range step.Status.Attributes {
			stepAttributes[k] = v
		}
	}
	workflow.Status.StepPhases = count
	workflow.Status.Attributes = stepAttributes
	if len(runErrors) > 0 {
		workflow.Status.RunError = strings.Join(runErrors, "\n")
	}
	if len(rollbackErrors) > 0 {
		workflow.Status.RollbackError = strings.Join(rollbackErrors, "\n")
	}
	if len(syncErrors) > 0 {
		workflow.Status.SyncError = strings.Join(syncErrors, "\n")
	}
	// 计算hash 要放在比较靠后的位置
	statusHash := calStatusHash(workflow)
	if statusHash != workflow.Status.Hash {
		// 仅触发一次，不管成功失败，都走下一步流程
		_ = r.onChange(ctx, workflow)
	}
	// 所有step 都成功了，则标记自己为成功
	if count[v1alpha1.StepSuccess] == len(steps) {
		workflow.Status.Phase = v1alpha1.WorkflowSuccess
		if currentPhase != v1alpha1.WorkflowSuccess {
			r.recorder.Eventf(workflow, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentPhase, workflow.Status.Phase)
		}
		return
	}
	// 有step失败，则标记为失败
	if count[v1alpha1.StepFailed] > 0 {
		workflow.Status.Phase = v1alpha1.WorkflowFailed
		if currentPhase != v1alpha1.WorkflowFailed {
			r.recorder.Eventf(workflow, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentPhase, workflow.Status.Phase)
		}
		return
	}
	// 所有step都回滚了，则标记回滚完成
	if count[v1alpha1.StepRollBacked] == len(steps) {
		workflow.Status.Phase = v1alpha1.WorkflowRollBacked
		if currentPhase != v1alpha1.WorkflowRollBacked {
			r.recorder.Eventf(workflow, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentPhase, workflow.Status.Phase)
		}
		return
	}
	// 有回滚step，则触发回滚。处理自发回滚的场景
	if count[v1alpha1.StepRollingBack] > 0 || count[v1alpha1.StepRollBacked] > 0 {
		workflow.Status.Phase = v1alpha1.WorkflowRollingBack
		if currentPhase != v1alpha1.WorkflowRollingBack {
			r.recorder.Eventf(workflow, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentPhase, workflow.Status.Phase)
		}
		return
	}
}

func (r *workflowReconciler) GetStepsForWorkflow(workflow *v1alpha1.Workflow) ([]v1alpha1.Step, error) {
	// Create selector.
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			"workflow": workflow.Name,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't convert Job selector: %v", err)
	}
	stepList := &v1alpha1.StepList{}
	err = r.client.List(context.Background(), stepList,
		client.MatchingLabelsSelector{Selector: selector}, client.InNamespace(workflow.GetNamespace()))
	if err != nil {
		return nil, err
	}
	return stepList.Items, nil
}

var seeAsRollBackedWorkflow = func(workflow *v1alpha1.Workflow) bool {
	// 都回滚完成了，才开始真正删除
	if workflow.Status.Phase == v1alpha1.WorkflowRollBacked {
		return true
	}
	if workflow.Spec.RollbackPolicy == v1alpha1.Always &&
		workflow.Status.StepPhases[v1alpha1.StepFailed]+workflow.Status.StepPhases[v1alpha1.StepRollBacked] == len(workflow.Spec.Steps) {
		return true
	}
	return false
}
var seeAsRollingBackWorkflow = func(workflow *v1alpha1.Workflow) bool {
	if workflow.Status.Phase == v1alpha1.WorkflowRollingBack {
		return true
	}
	rollbackCount := workflow.Status.StepPhases[v1alpha1.StepFailed] + workflow.Status.StepPhases[v1alpha1.StepRollBacked]
	if rollbackCount == 0 {
		return false
	}
	if workflow.Spec.RollbackPolicy == v1alpha1.Always && rollbackCount < len(workflow.Spec.Steps) {
		return true
	}
	return false
}

var calStatusHash = func(workflow *v1alpha1.Workflow) string {
	content := string(workflow.Status.Phase)
	for step, count := range workflow.Status.StepPhases {
		content += fmt.Sprintf("%s:%d", step, count)
	}
	for k, v := range workflow.Status.Attributes {
		content += fmt.Sprintf("%s:%s", k, v)
	}
	h := md5.New()
	if _, err := io.WriteString(h, content); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
