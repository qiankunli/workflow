## workflow

A workflow that natively supports rollback.

```
type Step interface {
	Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError // 或者叫healthcheck
}

```
the workflow controller will run `Step.Run` when it's upstream step is Success and run `Step.Rollback` when it's downstream step is Failed.

```
apiVersion: workflow.example.com/v1alpha1
kind: Workflow
metadata:
  name: example
spec:
  steps:
  - name: step1
    stepTemplate: 
      type: random
      parameters: 
        sleepSeconds: "20"
      retryPolicy:
        runRetryLimit: 3
        runRetryPeriodSeconds: 10
        rollbackRetryLimit: 3
        rollbackRetryPeriodSeconds: 3
  - name: step2
    dependOns:
    - name: step1
      phase: Success
    stepTemplate: 
      type: random
      parameters: 
        sleepSeconds: "20"
  - name: step3
    dependOns:
    - name: step2
      phase: Success
    stepTemplate: 
      type: random
      parameters: 
        sleepSeconds: "20"
```