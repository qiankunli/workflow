package operators

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	stepinterface "github.com/qiankunli/workflow/pkg/controller/step"
)

func (r *stepReconciler) reconcileRollback(_ context.Context, workflow *v1alpha1.Workflow, step *v1alpha1.Step) {
	log := r.log.WithValues("name", step.Name)
	s, err := stepinterface.NewStep(r.controllerCtx.Config, workflow, step)
	currentPhase := step.Status.Phase
	if err != nil {
		log.Error(err, "instantiate step error")
		step.Status.Phase = v1alpha1.StepFailed
		step.Status.RollbackError = err.Error()
		r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "'%s' => '%s',%v",
			currentPhase, v1alpha1.StepFailed, err)
		return
	}
	// 如果 RollbackRetryLimit <=0 则认为无限制重试
	if step.Spec.RetryPolicy.RollbackRetryLimit > 0 && step.Status.RollbackRetryCount >= step.Spec.RetryPolicy.RollbackRetryLimit {
		// 超过重试此处，则放弃，开始回滚
		step.Status.Phase = v1alpha1.StepFailed
		r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "'%s' => '%s',over RollbackRetryLimit",
			currentPhase, v1alpha1.StepFailed)
		return
	}
	r.runRollback(s, workflow, step)
}

func (r *stepReconciler) runRollback(s stepinterface.Step, workflow *v1alpha1.Workflow, step *v1alpha1.Step) {
	log := r.log.WithValues("name", step.Name)
	currentPhase := step.Status.Phase
	log.V(4).Info("run step rollback")
	stepErr := s.Rollback(workflow, step)
	if stepErr != nil && !stepErr.Ignorable() {
		step.Status.RollbackRetryCount++
	}
	step.Status.LatestRollbackRetryAt = metav1.Now()
	if stepErr != nil {
		log.Error(stepErr, "step rollback error")
		step.Status.RollbackError = stepErr.Error()
		if !stepErr.Retryable() {
			step.Status.Phase = v1alpha1.StepFailed
			r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "'%s' => '%s',rollbackRetryCount=%d error: %v",
				currentPhase, v1alpha1.StepFailed, step.Status.RollbackRetryCount, stepErr)
		} else {
			r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "rollbackRetryCount=%d error: %v", step.Status.RollbackRetryCount, stepErr)
		}
		return
	}
	// 清理掉之前可能的错误
	step.Status.RollbackError = ""
	step.Status.Phase = v1alpha1.StepRollBacked
	r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentPhase, v1alpha1.StepRollBacked)
}
