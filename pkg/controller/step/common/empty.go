package common

import (
	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	stepinterface "github.com/qiankunli/workflow/pkg/controller/step"
	"github.com/qiankunli/workflow/pkg/options"
)

func init() {
	stepinterface.Factory["empty"] = NewEmpty
}

type Empty struct {
}

func NewEmpty(cfg *options.Config, workflow *v1alpha1.Workflow, step *v1alpha1.Step) (stepinterface.Step, error) {
	return &Empty{}, nil
}

func (i *Empty) Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	return nil
}
func (i *Empty) Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	return nil

}

func (i *Empty) Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	return nil
}
