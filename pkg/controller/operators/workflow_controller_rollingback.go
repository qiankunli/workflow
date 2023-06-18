package operators

import (
	"context"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/utils/kube"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *workflowReconciler) reconcileRollingBack(ctx context.Context, workflow *v1alpha1.Workflow, steps []v1alpha1.Step) {
	log := r.log.WithValues("name", workflow.Name)
	// 下游step 回滚完成则触发下一个
	canRollingBackSteps := r.findRollingBackStep(workflow, steps)
	log.V(4).Info("find canRollingBackSteps", "count", len(canRollingBackSteps))
	for _, step := range canRollingBackSteps {
		curStep := step
		currentStepPhase := step.Status.Phase
		// 如果 任务本来就没有执行，可以考虑直接设置为 StepRollBacked
		if currentStepPhase == "" || currentStepPhase == v1alpha1.StepPending {
			log.V(4).Info("rollback step", "name", step.Name, "phase", currentStepPhase)
			base := step.DeepCopy()
			err := kube.RetryUpdateStatusOnConflict(ctx, r.client, base, func() error {
				base.Status.Phase = v1alpha1.StepRollBacked
				return nil
			})
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "update step error", "name", base.Name)
			} else {
				r.recorder.Eventf(&curStep, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentStepPhase, v1alpha1.StepRollBacked)
			}
		}
		if currentStepPhase == v1alpha1.StepRunning || currentStepPhase == v1alpha1.StepSuccess {
			log.V(4).Info("rollback step", "name", step.Name, "phase", currentStepPhase)
			base := step.DeepCopy()
			err := kube.RetryUpdateStatusOnConflict(ctx, r.client, base, func() error {
				base.Status.Phase = v1alpha1.StepRollingBack
				return nil
			})
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "update step error", "name", base.Name)
			} else {
				r.recorder.Eventf(&curStep, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'", currentStepPhase, v1alpha1.StepRollingBack)
			}
		}
	}
}

func (r *workflowReconciler) findRollingBackStep(workflow *v1alpha1.Workflow, steps []v1alpha1.Step) []v1alpha1.Step {
	stepMap := map[string]v1alpha1.Step{}
	for _, step := range steps {
		stepName := step.Labels["step"]
		stepMap[stepName] = step
	}
	// 建立反向依赖关系 <stepName,reverseDependOnStep>
	reverseDependOnMap := map[string][]string{}
	for _, workflowStep := range workflow.Spec.Steps {
		if len(workflowStep.DependOns) == 0 {
			continue
		}
		for _, dependOn := range workflowStep.DependOns {
			if len(reverseDependOnMap[dependOn.Name]) == 0 {
				reverseDependOnMap[dependOn.Name] = make([]string, 0)
			}
			reverseDependOnMap[dependOn.Name] = append(reverseDependOnMap[dependOn.Name], workflowStep.Name)
		}
	}
	ret := make([]v1alpha1.Step, 0)
	for _, step := range steps {
		stepName := step.Labels["step"]
		reverseDependOnSteps := reverseDependOnMap[stepName]
		// 没有反向依赖
		if len(reverseDependOnSteps) == 0 {
			ret = append(ret, stepMap[stepName])
			continue
		}
		count := 0
		for _, reverseDependOn := range reverseDependOnSteps {
			reverseDependOnStep, ok := stepMap[reverseDependOn]
			// 反向依赖step已不存在或已回滚
			if !ok || seeAsRollBackedStep(&reverseDependOnStep) {
				count++
			}
		}
		// 反向依赖step全部 StepRollBacked 或不存在 则本任务可以rollback
		if len(reverseDependOnSteps) == count {
			ret = append(ret, stepMap[stepName])
		}
	}
	return ret
}
