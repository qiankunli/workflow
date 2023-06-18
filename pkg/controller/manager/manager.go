package manager

import (
	"github.com/qiankunli/workflow/pkg/options"
	"github.com/qiankunli/workflow/pkg/utils/mutex"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ControllerContext ...
type ControllerContext struct {
	Config        *options.Config
	StepMutex     mutex.GroupMutex
	WorkflowMutex mutex.GroupMutex

	KubeClient kubernetes.Interface
}

// NewControllerContext ...
func NewControllerContext(restConfig *rest.Config, cfg *options.Config) (*ControllerContext, error) {
	ctrlClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &ControllerContext{
		Config:        cfg,
		KubeClient:    ctrlClient,
		WorkflowMutex: mutex.NewGroupMutex(),
		StepMutex:     mutex.NewGroupMutex(),
	}, nil
}
