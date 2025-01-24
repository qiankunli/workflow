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
	stepinterface.Factory["error"] = NewError
}

type Error struct {
	*v1alpha1.Step
}

func NewError(cfg *options.Config, workflow *v1alpha1.Workflow, step *v1alpha1.Step) (stepinterface.Step, error) {
	return &Error{
		Step: step,
	}, nil
}

type Data struct {
	// 模拟任务执行的耗时
	SleepSeconds int `json:"sleepSeconds,omitempty"`
}

func (i *Error) Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {

	sleepSeconds := 10
	if i.Step.Spec.Parameters != nil && len(i.Step.Spec.Parameters["sleepSeconds"]) > 0 {
		sleepSeconds = utils.ToInt(i.Step.Spec.Parameters["sleepSeconds"], 10)
	}
	time.Sleep(time.Duration(sleepSeconds) * time.Second)

	id := fmt.Sprintf("%d", rand.Int())
	if len(step.Status.Resource.ID) == 0 {
		step.Status.Resource.ID = id
		return stepinterface.NewCodeError("test", "run error", false, false)
	}
	return nil
}
func (i *Error) Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	sleepSeconds := 10
	if i.Step.Spec.Parameters != nil && len(i.Step.Spec.Parameters["sleepSeconds"]) > 0 {
		sleepSeconds = utils.ToInt(i.Step.Spec.Parameters["sleepSeconds"], 10)
	}

	time.Sleep(time.Duration(sleepSeconds) * time.Second)

	return stepinterface.NewCodeError("test", "rollback error", false, false)
}

func (i *Error) Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	return nil
}
