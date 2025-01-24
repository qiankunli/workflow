
English | [中文README](README.md)

## overview

Features
1. Native Support for Rollback 
   1. When a step fails, it triggers the rollback of previously successful steps in the workflow to ensure business consistency, which is the original motivation for this project.
2. Support for Step-Level Retries. Includes retries for both step tasks and step rollbacks, with support for configurable retry intervals.
3. Support for Step-Level Rate Limiting
4. Support for Asynchronous and Long-Running Steps
5. Custom Step Implementation via Go Code. To customize step implementations, you need to fork the project and integrate the code into your own project. In the future, we may consider providing some pre-configured general step implementations.
6. Support for Parallel Execution. A workflow can consist of multiple steps, with configurable dependencies between steps. Steps without dependencies can run in parallel, while others run sequentially. Rollbacks follow the same rules.
7. Data Exchange Between Steps. Steps can exchange data through the workflow to collaborate and complete tasks.
8. Multi-Tenancy Support. Each workflow has a spec.queue, and workflows from different queues are processed fairly.
9. Callback Support. Notifications can be sent to the business side via callback URLs when a workflow starts, succeeds, fails, or when a step succeeds or fails.

## Installation 

configure `manifest/workflow-controller/templates/configmap`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-config
  namespace: {{ .Release.Namespace | quote }}
data:
  config.yaml: |
    namespace: {{ .Release.Namespace }}
    throttleQPS: 100
    idleTimeoutMillSeconds: 200000
    rpcTimeoutMillSeconds: 5000
    controllerConfig:
      queue:
        strategy: "FIFO"       
        maxRunningCount: 100   
      steps:
      - kind: "random"
        qps: 1                 // Limit the consumption rate of a certain type of step.

```
install controller
```
kubectl apply -f manifest/workflow-controller/crd.yaml
helm install workflow -f manifest/workflow-controller
```

## step definition

For each step, it must conform to the following Go interface specification.
```
type Step interface {
	Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError // 或者叫healthcheck
}
```

## workflow definition

The Workflow controller will:
1. Trigger `step.Run` when the upstream steps of the step have executed successfully.
2. Trigger `step.Rollback` when the downstream steps of the step have executed unsuccessfully.
3. Trigger `step.Sync` periodically to update the step's status.

```
apiVersion: workflow.example.com/v1alpha1
kind: Workflow
metadata:
  name: example
spec:
  queue: default
  callback: 
    url:  http://locahost:8080/abc
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
    stepTemplate: 
      type: random          # random is a demo implementent of step interface
      parameters: 
        sleepSeconds: "20"
```


When the Workflow execution status changes, an HTTP callback will be triggered. The interface details are as follows:

```
curl -X POST http://locahost:8080/abc
-d '{
   "name": xx // workflow name
   "phase": xx // workflow phase
   "attributes": xx // workflow attributes
   "runError": xx // workflow/step run error
   "rollbackError": xx // workflow/step run error
}'
```