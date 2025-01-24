[ENGLISH README](README_en.md)| 中文

## 关于 workflow

workflow 是一个基于k8s crd 实现的工作流引擎，一个workflow 由多个step 组成，每个step 代表一个原子操作。

特点

1. 原生支持rollback
    1. 当某个step执行失败时，触发workflow中已执行成功的step回滚，进而确保业务的一致性，这也是本项目发起的初衷
2. 支持step 级别的重试，包括step 任务的重试，和step 回滚的重试，支持重试间隔
3. 支持step 级别的限流
4. 对于异步、长耗时step
5. 通过go 代码来定制step实现。为此你需要fork 该项目，并将代码集成到你的项目中，未来会考虑预置一些通用step实现
6. 一个workflow 可以由多个step组成，可以配置step的依赖关系，无依赖关系的step可以并发执行，否则串行执行，回滚时也是。
7. step之间可以通过workflow来交换数据，协作完成任务
8. 支持多租户，每一个workflow 有一个spec.queue，支持不同queue之间的workflow公平消费
9. 支持回调，当workflow 开始执行、执行成功、执行失败、step执行成功、失败时，可以通过回调url来通知业务方

## 安装

调整manifest/workflow-controller/templates/configmap

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
        strategy: "FIFO"       // workflow 按创建时间消费，也可以设置为FAIR，按queue 公平消费
        maxRunningCount: 100   // 可以同时运行的workflow 数量
      steps:
      - kind: "random"
        qps: 1                 // 限制某个类型的step的消费速度

```

```
kubectl apply -f manifest/workflow-controller/crd.yaml
helm install workflow -f manifest/workflow-controller
```
fork 项目后添加自定义业务step实现
```
workflow
   /pkg
      /controller
      /step             // 可以在该目录下自定义自己的step实现
         /example       // 示例step的实现
            /random.go  
         /interface.go  // step 接口定义
```

## step定义

对于每一个step，必须符合以下golang interface 规范
```
type Step interface {
	Run(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Rollback(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError
	Sync(workflow *v1alpha1.Workflow, step *v1alpha1.Step) StepError // 或者叫healthcheck
}
```


## workflow 定义

workflow controller 将会
1. 触发`step.Run` 当step上游的step 执行成功
2. 触发`step.Rollback` 当step下游的step 执行失败
3. 每隔一段时间触发`step.Sync`，用来变更step.status

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

workflow 运行状态变更时会触发http callback，接口详情如下

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