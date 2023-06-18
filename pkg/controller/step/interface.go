package internal

import (
	"fmt"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
)

type StepError interface {
	Error() string
	Retryable() bool
}
type stepError struct {
	error     error
	retryable bool
}

func (s *stepError) Error() string {
	return s.error.Error()
}
func (s *stepError) Retryable() bool {
	return s.retryable
}

type codeError struct {
	code      string
	message   string
	retryable bool
}

func (s *codeError) Error() string {
	return fmt.Sprintf("code: %s, message: %s", s.code, s.message)
}
func (s *codeError) Retryable() bool {
	return s.retryable
}

func NewCodeError(code, message string, retryable bool) StepError {
	return &codeError{
		code:      code,
		message:   message,
		retryable: retryable,
	}
}

func NewStepError(err error, retryable bool) StepError {
	return &stepError{
		error:     err,
		retryable: retryable,
	}
}

type Step interface {
	Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError // 或者叫healthcheck
}
