package internal

import (
	"fmt"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	"github.com/qiankunli/workflow/pkg/options"
	"k8s.io/klog/v2"
)

type NewStepFunc func(cfg *options.Config, workflow *v1alpha1.Workflow, step *v1alpha1.Step) (Step, error)

var Factory = map[string]NewStepFunc{}

func NewStep(cfg *options.Config, workflow *v1alpha1.Workflow, step *v1alpha1.Step) (Step, error) {
	newFunc, ok := Factory[step.Spec.Type]
	if !ok {
		err := fmt.Errorf("can not find step type %s", step.Spec.Type)
		klog.ErrorS(err, "can not find step type")
		return nil, err
	}
	return newFunc(cfg, workflow, step)
}
