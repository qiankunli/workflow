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
	stepinterface.Factory["random"] = NewRandom
}

type Random struct {
	*v1alpha1.Step
}

func NewRandom(cfg *options.Config, workflow *v1alpha1.Workflow, step *v1alpha1.Step) (stepinterface.Step, error) {
	return &Random{
		Step: step,
	}, nil
}

func (i *Random) Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	sleepSeconds := 10
	if i.Step.Spec.Parameters != nil && len(i.Step.Spec.Parameters["sleepSeconds"]) > 0 {
		sleepSeconds = utils.ToInt(i.Step.Spec.Parameters["sleepSeconds"], 10)
	}

	time.Sleep(time.Duration(sleepSeconds) * time.Second)

	id := fmt.Sprintf("%d", rand.Int())
	step.Status.Resource.ID = id
	return nil
}
func (i *Random) Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {

	sleepSeconds := 10
	if i.Step.Spec.Parameters != nil && len(i.Step.Spec.Parameters["sleepSeconds"]) > 0 {
		sleepSeconds = utils.ToInt(i.Step.Spec.Parameters["sleepSeconds"], 10)
	}

	time.Sleep(time.Duration(sleepSeconds) * time.Second)
	return nil

}

func (i *Random) Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) stepinterface.StepError {
	// 更新step 可能引起workflow 更新step 冲突
	if rand.Int()%2 == 0 {
		id := fmt.Sprintf("%d", rand.Int())
		step.Status.Resource.ID = id
		step.Status.Attributes[step.Spec.Type] = id
	}
	return nil
}
