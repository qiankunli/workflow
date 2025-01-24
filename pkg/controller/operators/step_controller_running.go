package operators

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	stepinterface "github.com/qiankunli/workflow/pkg/controller/step"
)

func (r *stepReconciler) reconcileRun(_ context.Context, workflow *v1alpha1.Workflow, step *v1alpha1.Step) {
	log := r.log.WithValues("name", step.Name)
	s, err := stepinterface.NewStep(r.controllerCtx.Config, workflow, step)
	currentPhase := step.Status.Phase
	if err != nil {
		log.Error(err, "instantiate step error")
		step.Status.Phase = v1alpha1.StepFailed
		step.Status.RunError = err.Error()
		r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "'%s' => '%s'%v",
			currentPhase, v1alpha1.StepFailed, err)
		return
	}

	if step.Status.RunRetryCount >= step.Spec.RetryPolicy.RunRetryLimit {
		// 超过重试此处，则放弃，开始回滚
		step.Status.Phase = v1alpha1.StepRollingBack
		r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s',over RunRetryLimit",
			currentPhase, step.Status.Phase)
		return
	}
	r.runRun(s, workflow, step)
}

func (r *stepReconciler) runRun(s stepinterface.Step, workflow *v1alpha1.Workflow, step *v1alpha1.Step) {
	log := r.log.WithValues("name", step.Name)
	currentPhase := step.Status.Phase

	//  准备好以便 step func 使用
	if step.Status.Resource.Attributes == nil {
		step.Status.Resource.Attributes = map[string]string{}
	}

	log.V(4).Info("run step run")
	stepErr := s.Run(workflow, step)
	step.Status.RunRetryCount++
	step.Status.LatestRunRetryAt = metav1.Now()
	if stepErr != nil {
		log.Error(stepErr, "step run error")
		step.Status.RunError = stepErr.Error()
		if !stepErr.Retryable() {
			// 发现不可重试的错误，立即触发回滚
			step.Status.Phase = v1alpha1.StepRollingBack
			r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "'%s' => '%s',runRetryCount=%d error: %v",
				currentPhase, v1alpha1.StepRollingBack, step.Status.RunRetryCount, stepErr)
		} else {
			r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.FailedOrErrorReason, "runRetryCount=%d error: %v", step.Status.RunRetryCount, stepErr)
		}
		return
	}
	// 清理掉之前可能的错误
	step.Status.RunError = ""
	step.Status.Phase = v1alpha1.StepSuccess
	r.recorder.Eventf(step, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentPhase, step.Status.Phase)
}
