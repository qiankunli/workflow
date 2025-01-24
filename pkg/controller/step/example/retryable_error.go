package example

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	stepinterface "github.com/qiankunli/workflow/pkg/controller/step"
	"github.com/qiankunli/workflow/pkg/options"
	"github.com/qiankunli/workflow/pkg/utils"
)

func init() {
	stepinterface.Factory["retryable_error"] = NewRetryableError
}

type RetryableError struct {
	*v1alpha1.Step
}

func NewRetryableError(cfg *options.Config, workflow *v1alpha1.Workflow, step *v1alpha1.Step) (stepinterface.Step, error) {
	return &RetryableError{
		Step: step,
	}, nil
}

func (i *RetryableError) Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	sleepSeconds := 10
	if i.Step.Spec.Parameters != nil && len(i.Step.Spec.Parameters["sleepSeconds"]) > 0 {
		sleepSeconds = utils.ToInt(i.Step.Spec.Parameters["sleepSeconds"], 10)
	}
	time.Sleep(time.Duration(sleepSeconds) * time.Second)

	id := fmt.Sprintf("%d", rand.Int())
	if len(step.Status.Resource.ID) == 0 {
		step.Status.Resource.ID = id
		return stepinterface.NewCodeError("test", "first return error", true, false)
	}
	step.Status.Resource.ID = id
	return nil
}
func (i *RetryableError) Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {

	sleepSeconds := 10
	if i.Step.Spec.Parameters != nil && len(i.Step.Spec.Parameters["sleepSeconds"]) > 0 {
		sleepSeconds = utils.ToInt(i.Step.Spec.Parameters["sleepSeconds"], 10)
	}
	time.Sleep(time.Duration(sleepSeconds) * time.Second)

	return nil

}

func (i *RetryableError) Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	return nil
}
