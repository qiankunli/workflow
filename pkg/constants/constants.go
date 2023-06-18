package constants

import "time"

// for controller
const (
	WorkflowPrefix         = "workflow.example.com"
	FinalizersWorkflow     = WorkflowPrefix + "/workflow-finalizers"
	DefaultRequeueDuration = 10 * time.Second
)
