package internal

import (
	"fmt"

	"github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
)

type StepError interface {
	Error() string
	Retryable() bool
	Ignorable() bool
}
type stepError struct {
	error     error
	retryable bool
	ignorable bool
}

func (s *stepError) Error() string {
	return s.error.Error()
}
func (s *stepError) Retryable() bool {
	return s.retryable
}
func (s *stepError) Ignorable() bool {
	return s.ignorable
}

type codeError struct {
	code      string
	message   string
	retryable bool
	// 一些错误比如限流，算执行失败，但是可以忽略的，比如限流，不应计入重试次数
	ignorable bool
}

func (s *codeError) Error() string {
	return fmt.Sprintf("code: %s, message: %s", s.code, s.message)
}
func (s *codeError) Retryable() bool {
	return s.retryable
}
func (s *codeError) Ignorable() bool {
	return s.ignorable
}

func NewCodeError(code, message string, retryable, ignorable bool) StepError {
	return &codeError{
		code:      code,
		message:   message,
		retryable: retryable,
		ignorable: ignorable,
	}
}

func NewStepError(err error, retryable, ignorable bool) StepError {
	return &stepError{
		error:     err,
		retryable: retryable,
		ignorable: ignorable,
	}
}

type Step interface {
	Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError // 或者叫healthcheck
}
