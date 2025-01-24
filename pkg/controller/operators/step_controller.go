package operators

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	k8sutilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/constants"
	"github.com/qiankunli/workflow/pkg/controller/manager"
	stepinterface "github.com/qiankunli/workflow/pkg/controller/step"
	controlleroptions "github.com/qiankunli/workflow/pkg/options/controller"
	"github.com/qiankunli/workflow/pkg/utils"
	"github.com/qiankunli/workflow/pkg/utils/kube"
	"github.com/qiankunli/workflow/pkg/utils/mutex"

	// 引入example step
	_ "github.com/qiankunli/workflow/pkg/controller/step/example"
	// 引入common step
	_ "github.com/qiankunli/workflow/pkg/controller/step/common"
)

type stepReconciler struct {
	client client.Client

	controllerCtx *manager.ControllerContext
	log           logr.Logger
	recorder      record.EventRecorder
	StepMutex     mutex.GroupMutex
}

// RegisterStepReconciler ...
func RegisterStepReconciler(mgr ctrl.Manager, controllerCtx *manager.ControllerContext, stepConfig controlleroptions.StepConfig) error {
	const name = "step-controller"

	r := &stepReconciler{
		client:        mgr.GetClient(),
		controllerCtx: controllerCtx,
		log:           ctrl.LoggerFrom(context.Background()).WithName(name),
		recorder:      mgr.GetEventRecorderFor(name),
		StepMutex:     controllerCtx.StepMutex,
	}

	// 只执行特性类型的step
	stepPredicateFn := func(object client.Object) bool {
		c, ok := object.(*v1alpha1.Step)
		if !ok {
			return false
		}
		return stepConfig.Kind == c.Spec.Type
	}
	stepPredicate := builder.WithPredicates(predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return stepPredicateFn(e.Object) },
		UpdateFunc:  func(e event.UpdateEvent) bool { return stepPredicateFn(e.ObjectNew) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return stepPredicateFn(e.Object) },
		GenericFunc: func(e event.GenericEvent) bool { return stepPredicateFn(e.Object) },
	})
	_, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			CacheSyncTimeout:        utils.FirstNotZeroDuration(stepConfig.SyncTimeout, controllerCtx.Config.ControllerConfig.SyncTimeout).Duration,
			MaxConcurrentReconciles: utils.FirstNotZeroInt(stepConfig.Concurrency, controllerCtx.Config.ControllerConfig.Concurrency),
			RateLimiter: workqueue.NewMaxOfRateLimiter(
				workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
				&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(stepConfig.Qps), 100)},
			),
		}).
		For(&v1alpha1.Step{}, stepPredicate).
		Named(name).
		Build(r)

	if err != nil {
		return fmt.Errorf("failed to set up with manager: %w", err)
	}
	r.log.Info("succeeded to set up with manager")
	return nil
}

// Reconcile ...
func (r *stepReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, reterr error) {
	log := r.log.WithValues("name", req.Name)
	defer func() {
		if p := recover(); p != nil {
			log.Error(errors.Errorf("panic error: %+v", p), "Panic error")
			debug.PrintStack()
		}
	}()

	// Only one reconcile routine can deal for per cr
	lockKey := req.Namespace + req.Name
	if !r.StepMutex.Lock(lockKey) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	defer r.StepMutex.Unlock(lockKey)

	ctx = ctrl.LoggerInto(ctx, log)
	log.V(4).Info("step start reconcile")

	// Fetch the step
	step := &v1alpha1.Step{}
	if err := r.client.Get(ctx, req.NamespacedName, step); err != nil {
		if k8sapierrors.IsNotFound(err) {
			log.Info("step has been deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get step")
		return ctrl.Result{}, err
	}
	log.V(4).Info("step start reconcile", "phase", step.Status.Phase)
	// Initialize the patch helper, defer patch the step
	stepPatchHelper, err := kube.NewHelper(step, r.client)
	if err != nil {
		log.Error(err, "new step patchHelper error")
		return ctrl.Result{}, err
	}
	defer func() {
		if err := stepPatchHelper.Patch(ctx, step); err != nil {
			reterr = k8sutilerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// workflow own step，从日志上看，workflow 被完全删除时，step  DeletionTimestamp 开始不为空
	if !step.DeletionTimestamp.IsZero() {
		log.V(4).Info("step deletionTimestamp is not zero", "phase", step.Status.Phase)
		// 回滚完成了，才能删除
		if seeAsRollBackedStep(step) {
			controllerutil.RemoveFinalizer(step, constants.FinalizersWorkflow)
			r.StepMutex.DelMutex(lockKey)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}

	if !controllerutil.ContainsFinalizer(step, constants.FinalizersWorkflow) {
		controllerutil.AddFinalizer(step, constants.FinalizersWorkflow)
	}

	// Fetch the workflow, workflow.DeletionTimestamp 不为空时，依然可以查到
	workflow := &v1alpha1.Workflow{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: step.Labels["workflow"]}, workflow); err != nil {
		if k8sapierrors.IsNotFound(err) {
			log.Info("workflow has been deleted, terminate reconcile", "name", step.Labels["workflow"])
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get workflow", "workflow name", step.Labels["workflow"])
		return ctrl.Result{}, err
	}
	workflowPatchHelper, err := kube.NewHelper(workflow, r.client)
	if err != nil {
		log.Error(err, "new workflow patchHelper error")
		return ctrl.Result{}, err
	}
	defer func() {
		if err := workflowPatchHelper.Patch(ctx, workflow); err != nil {
			reterr = k8sutilerrors.NewAggregate([]error{reterr, err})
		}
	}()
	log.V(4).Info("step start reconcile", "workflow.Phase", workflow.Status.Phase, "workflow.DeletionTimestamp", workflow.DeletionTimestamp)

	if step.Status.Phase == v1alpha1.StepRunning {
		log.V(4).Info("try run step run", "LatestRunRetryAt", step.Status.LatestRunRetryAt)
		needWaitDuration := time.Duration(step.Spec.RetryPolicy.RunRetryPeriodSeconds) * time.Second
		if !step.Status.LatestRunRetryAt.IsZero() {
			nextRunAt := step.Status.LatestRunRetryAt.Time.Add(needWaitDuration)
			needWaitDuration = time.Until(nextRunAt)
			if needWaitDuration > 0 {
				// 没到执行时间
				return ctrl.Result{RequeueAfter: needWaitDuration}, nil
			}
		}
		// 到了执行时间
		r.reconcileRun(ctx, workflow, step)
		// 没成功下次继续
		if step.Status.Phase == v1alpha1.StepRunning {
			return ctrl.Result{RequeueAfter: needWaitDuration}, nil
		}
		// 成功则进入Success，还需sync，所以过一会儿入队
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}
	if step.Status.Phase == v1alpha1.StepRollingBack {
		log.V(4).Info("try run step rollback", "LatestRollbackRetryAt", step.Status.LatestRollbackRetryAt)
		needWaitDuration := time.Duration(step.Spec.RetryPolicy.RollbackRetryPeriodSeconds) * time.Second
		if !step.Status.LatestRollbackRetryAt.IsZero() {
			nextRollbackAt := step.Status.LatestRollbackRetryAt.Time.Add(needWaitDuration)
			needWaitDuration = time.Until(nextRollbackAt)
			if needWaitDuration > 0 {
				// 没到执行时间
				return ctrl.Result{RequeueAfter: needWaitDuration}, nil
			}
		}
		r.reconcileRollback(ctx, workflow, step)
		// 没成功下次继续
		if step.Status.Phase == v1alpha1.StepRollingBack {
			return ctrl.Result{RequeueAfter: needWaitDuration}, nil
		}
		// 成功则进入RollBacked
		return ctrl.Result{}, nil
	}
	if step.Status.Phase == v1alpha1.StepSuccess {
		r.reconcileSync(ctx, workflow, step)
		// 因为要sync，所以要一会儿再进来看下
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}
	return ctrl.Result{}, nil
}

var seeAsRollBackedStep = func(step *v1alpha1.Step) bool {
	// 都回滚完成了，才开始真正删除
	if step.Status.Phase == v1alpha1.StepRollBacked {
		return true
	}
	if step.Status.Phase == v1alpha1.StepFailed && step.Spec.RollbackPolicy == v1alpha1.Always {
		return true
	}
	return false
}

func (r *stepReconciler) reconcileSync(_ context.Context, workflow *v1alpha1.Workflow, step *v1alpha1.Step) {
	log := r.log.WithValues("name", step.Name)
	currentPhase := step.Status.Phase
	s, err := stepinterface.NewStep(r.controllerCtx.Config, workflow, step)
	if err != nil {
		log.Error(err, "instantiate step error")
		return
	}
	stepErr := s.Sync(workflow, step)
	if stepErr != nil {
		log.Error(stepErr, "step sync error")
		step.Status.RunError = stepErr.Error()
		if !stepErr.Retryable() {
			// 发现不可重试的错误，立即触发回滚
			step.Status.Phase = v1alpha1.StepRollingBack
			r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "'%s' => '%s',sync error: %v", currentPhase, v1alpha1.StepRollingBack, stepErr)
		}
	}
}
