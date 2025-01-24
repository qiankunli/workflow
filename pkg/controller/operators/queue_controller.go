package operators

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sutilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/constants"
	"github.com/qiankunli/workflow/pkg/controller/manager"
	controller2 "github.com/qiankunli/workflow/pkg/options/controller"
	"github.com/qiankunli/workflow/pkg/utils"
	"github.com/qiankunli/workflow/pkg/utils/kube"
)

type queueReconciler struct {
	client client.Client

	controllerCtx   *manager.ControllerContext
	queueStrategy   controller2.QueueStrategy
	maxRunningCount int
	log             logr.Logger
	recorder        record.EventRecorder
}

// RegisterQueueReconciler ...
func RegisterQueueReconciler(mgr ctrl.Manager, controllerCtx *manager.ControllerContext) error {
	const name = "queue-controller"

	r := &queueReconciler{
		client:          mgr.GetClient(),
		controllerCtx:   controllerCtx,
		log:             ctrl.LoggerFrom(context.Background()).WithName(name),
		recorder:        mgr.GetEventRecorderFor(name),
		maxRunningCount: controllerCtx.Config.ControllerConfig.Queue.MaxRunningCount,
		queueStrategy:   controllerCtx.Config.ControllerConfig.Queue.Strategy,
	}

	_, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			CacheSyncTimeout:        controllerCtx.Config.ControllerConfig.SyncTimeout.Duration,
			MaxConcurrentReconciles: 1,
		}).
		For(&v1alpha1.Workflow{}).
		Named(name).
		Build(r)

	if err != nil {
		return fmt.Errorf("failed to set up with manager: %w", err)
	}
	r.log.Info("succeeded to set up with manager")
	return nil
}

// Reconcile ...
func (r *queueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, reterr error) {
	log := r.log.WithValues("name", req.Name)
	defer func() {
		if p := recover(); p != nil {
			log.Error(errors.Errorf("panic error: %+v", p), "Panic error")
			debug.PrintStack()
		}
	}()

	ctx = ctrl.LoggerInto(ctx, log)
	log.V(4).Info("queue start reconcile")

	workflowList := &v1alpha1.WorkflowList{}
	if err := r.client.List(context.Background(), workflowList); err != nil {
		log.Error(err, "failed to list workflow")
		return ctrl.Result{}, err
	}
	if len(workflowList.Items) > r.maxRunningCount {
		log.V(4).Info(fmt.Sprintf("running workflow limit exceeded maxRunning count: %d", r.maxRunningCount))
		return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
	}
	var workflow *v1alpha1.Workflow
	if r.queueStrategy == controller2.FIFO {
		workflow = &v1alpha1.Workflow{}
		if err := r.client.Get(ctx, req.NamespacedName, workflow); err != nil {
			if k8sapierrors.IsNotFound(err) {
				klog.InfoS("workflow has been deleted")
				return ctrl.Result{}, err
			}
			log.Error(err, "failed to get workflow")
			return ctrl.Result{}, err
		}
	} else if r.queueStrategy == controller2.FIFO {
		workflow = findNeedRunningWorkflow(workflowList)
	}
	if workflow != nil {
		patchHelper, err := kube.NewHelper(workflow, r.client)
		if err != nil {
			return ctrl.Result{}, err
		}
		defer func() {
			if err := patchHelper.Patch(ctx, workflow); err != nil {
				reterr = k8sutilerrors.NewAggregate([]error{reterr, err})
			}
		}()
		if workflow.Status.Phase == v1alpha1.WorkflowPending {
			workflow.Status.Phase = v1alpha1.WorkflowRunning
			log.V(4).Info(fmt.Sprintf("trigger queue %s workflow %s running", workflow.Spec.Queue, workflow.Name))
			r.recorder.Eventf(workflow, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%v' => '%s'",
				utils.FirstNotNull(v1alpha1.WorkflowPending, v1alpha1.WorkflowPending), workflow.Status.Phase)
		}
	}
	return ctrl.Result{RequeueAfter: constants.DefaultRequeueDuration}, nil
}

func findNeedRunningWorkflow(workflowList *v1alpha1.WorkflowList) *v1alpha1.Workflow {
	queueWorkflowRunningCount := map[string]int{}
	queueWorkflowPendingCount := map[string]int{}
	queueWorkflow := map[string][]v1alpha1.Workflow{}
	for _, workflow := range workflowList.Items {
		if wfList, ok := queueWorkflow[workflow.Spec.Queue]; ok {
			wfList = append(wfList, workflow)
		} else {
			queueWorkflow[workflow.Spec.Queue] = []v1alpha1.Workflow{workflow}
		}
		if workflow.Status.Phase == v1alpha1.WorkflowRunning {
			queueWorkflowRunningCount[workflow.Spec.Queue]++
		} else if workflow.Status.Phase == v1alpha1.WorkflowPending {
			queueWorkflowPendingCount[workflow.Spec.Queue]++
		}
	}
	// 找到running 数量最少 且有pending workflow 的queue
	needRunningQueue := ""
	minWorkflowRunningCount := 1
	for queue, _ := range queueWorkflow {
		if queueWorkflowPendingCount[queue] > 0 {
			if queueWorkflowRunningCount[queue] < minWorkflowRunningCount {
				minWorkflowRunningCount = queueWorkflowRunningCount[queue]
				needRunningQueue = queue
			}
		}
	}
	var needRunningWorkflow *v1alpha1.Workflow
	if len(needRunningQueue) > 0 {
		needRunningWorkflows := queueWorkflow[needRunningQueue]
		minCreateTime := needRunningWorkflows[0].CreationTimestamp
		needRunningWorkflow = &needRunningWorkflows[0]
		for _, workflow := range needRunningWorkflows {
			if workflow.CreationTimestamp.Before(&minCreateTime) {
				minCreateTime = workflow.CreationTimestamp
				needRunningWorkflow = &workflow
			}
		}
	}
	return needRunningWorkflow
}
