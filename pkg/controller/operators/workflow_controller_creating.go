package operators

import (
	"context"
	"fmt"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *workflowReconciler) reconcileCreating(ctx context.Context, workflow *v1alpha1.Workflow, steps []v1alpha1.Step) {
	currentPhase := workflow.Status.Phase
	// 去重
	stepSet := make(map[string]bool, 0)
	for _, s := range steps {
		stepSet[s.Name] = true
	}
	for _, ws := range workflow.Spec.Steps {
		if stepSet[ws.Name] {
			continue
		}
		if err := r.createStep(ctx, workflow, ws); err != nil {
			return
		}
	}
	workflow.Status.Phase = v1alpha1.WorkflowRunning
	r.recorder.Eventf(workflow, corev1.EventTypeNormal, v1alpha1.PhaseChangeReason, "'%v' => '%s'",
		utils.FirstNotNull(currentPhase, v1alpha1.WorkflowPending), workflow.Status.Phase)
}

func (r *workflowReconciler) createStep(ctx context.Context, workflow *v1alpha1.Workflow, ws v1alpha1.WorkflowStep) error {
	log := r.log.WithValues("name", workflow.Name)

	step := &v1alpha1.Step{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: workflow.Namespace,
			// 不同 workflow 的step 名称应避免重复
			Name:            fmt.Sprintf("%s-%s", workflow.Name, ws.Name),
			OwnerReferences: []metav1.OwnerReference{*GenOwnerReference(workflow)},
			Labels: map[string]string{
				"workflow": workflow.Name,
				"step":     ws.Name,
			},
		},
		Spec: ws.StepTemplate,
		Status: v1alpha1.StepStatus{
			Phase: v1alpha1.StepPending,
		},
	}
	// step RollbackPolicy 默认与workflow 保持一致
	step.Spec.RollbackPolicy = workflow.Spec.RollbackPolicy

	if err := r.client.Create(ctx, step); err != nil {
		log.Error(err, "create step error", "name", step.Name)
		return err
	}
	log.V(3).Info("create step success", "name", step.Name)
	return nil
}

func GenOwnerReference(obj metav1.Object) *metav1.OwnerReference {
	boolPtr := func(b bool) *bool { return &b }
	controllerRef := &metav1.OwnerReference{
		APIVersion:         v1alpha1.GroupVersion.String(),
		Kind:               "Workflow",
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}
