package operators

import (
	"context"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/utils"
	"github.com/qiankunli/workflow/pkg/utils/kube"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *workflowReconciler) reconcileRunning(ctx context.Context, workflow *v1alpha1.Workflow, steps []v1alpha1.Step) {
	log := r.log.WithValues("name", workflow.Name)
	// 有step 成功，则触发下一个
	canRunningSteps := r.findCanRunningStep(workflow, steps)
	log.V(4).Info("find canRollingBackSteps", "count", len(canRunningSteps))
	for _, step := range canRunningSteps {
		curStep := step
		currentStepPhase := step.Status.Phase
		if currentStepPhase == "" || currentStepPhase == v1alpha1.StepPending {
			log.V(4).Info("change step running", "name", step.Name)
			base := step.DeepCopy()
			err := kube.RetryUpdateStatusOnConflict(ctx, r.client, base, func() error {
				base.Status.Phase = v1alpha1.StepRunning
				return nil
			})
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "update step error", "name", base.Name)
			} else {
				r.recorder.Eventf(&curStep, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%s' => '%s'",
					utils.FirstNotNull(currentStepPhase, v1alpha1.StepPending), v1alpha1.StepRunning)
			}
		}
	}
}

func (r *workflowReconciler) findCanRunningStep(workflow *v1alpha1.Workflow, steps []v1alpha1.Step) []v1alpha1.Step {
	stepMap := map[string]v1alpha1.Step{}
	for _, step := range steps {
		stepName := step.Labels["step"]
		stepMap[stepName] = step
	}
	ret := make([]v1alpha1.Step, 0)
	for _, workflowStep := range workflow.Spec.Steps {
		// 判断 当前 workflowStep 是否可以running
		// 没有依赖，可以执行
		if len(workflowStep.DependOns) == 0 {
			ret = append(ret, stepMap[workflowStep.Name])
			continue
		}
		dependOnCount := 0
		for _, dependOn := range workflowStep.DependOns {
			dependOnStep := stepMap[dependOn.Name]
			dependOnStepPhase := dependOnStep.Status.Phase
			// 依赖step 的phase 不对
			if dependOn.Phase != dependOnStepPhase {
				continue
			}
			// 依赖step 的ResourceStatus 不对，如果有的话
			if len(dependOn.ResourceStatus) > 0 {
				dependOnStepResourceStatus := dependOnStep.Status.Resource.Status
				if dependOn.ResourceStatus != dependOnStepResourceStatus {
					continue
				}
			}
			dependOnCount++
		}
		// 依赖的任务全部进入指定状态
		if dependOnCount >= len(workflowStep.DependOns) {
			ret = append(ret, stepMap[workflowStep.Name])
		}
	}
	return ret
}
