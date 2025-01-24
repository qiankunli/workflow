/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Queue string `json:"queue,omitempty"`
	// +kubebuilder:default:=PreserveOnFailure
	RollbackPolicy RollbackPolicy `json:"rollbackPolicy,omitempty"`
	// Map类型的数据
	Parameters map[string]string `json:"parameters,omitempty"`
	Steps      []WorkflowStep    `json:"steps,omitempty"`
}

type DependOn struct { // 描述该step 依赖其他step的情况
	// step
	Name string `json:"name,omitempty"`

	Phase StepPhase `json:"phase,omitempty"`
	// 依赖step resource进入xx 状态
	ResourceStatus string `json:"resourceStatus,omitempty"`
}

type WorkflowStep struct {
	Name         string     `json:"name,omitempty"`
	DependOns    []DependOn `json:"dependOns,omitempty"`
	StepTemplate StepSpec   `json:"stepTemplate,omitempty"`
}

// RollbackPolicy
// +kubebuilder:validation:Enum=Always;PreserveOnFailure
type RollbackPolicy string

const (
	//Always 删除workflow时，哪怕step 有failed 也要继续删
	Always RollbackPolicy = "Always"
	//PreserveOnFailure 删除workflow时，step 有failed 则中断，保留现场
	PreserveOnFailure RollbackPolicy = "PreserveOnFailure"
)

// WorkflowPhase
// +kubebuilder:validation:Enum=Pending;Running;Success;RollingBack;RollBacked;Failed
type WorkflowPhase string

const (
	WorkflowPending     WorkflowPhase = "Pending"
	WorkflowRunning     WorkflowPhase = "Running"
	WorkflowSuccess     WorkflowPhase = "Success"
	WorkflowRollingBack WorkflowPhase = "RollingBack"
	WorkflowRollBacked  WorkflowPhase = "RollBacked"
	WorkflowFailed      WorkflowPhase = "Failed"
)

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default:=Pending
	Phase      WorkflowPhase     `json:"phase,omitempty"`
	StepPhases map[StepPhase]int `json:"stepPhases,omitempty"`

	Attributes map[string]string `json:"attributes,omitempty"`
	RunError   string            `json:"runError,omitempty"`
}

// Workflow is the Schema for the workflows API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=workflows,shortName=wf,scope=Namespaced
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="workflow phase. "
// +kubebuilder:printcolumn:name="RunningSteps",type="integer",JSONPath=".status.stepPhases.Running",description="running step count"
// +kubebuilder:printcolumn:name="SuccessSteps",type="integer",JSONPath=".status.stepPhases.Success",description="success step count"
// +kubebuilder:printcolumn:name="RollingBackSteps",type="integer",JSONPath=".status.stepPhases.RollingBack",description="rollingBack step count"
// +kubebuilder:printcolumn:name="RollBackedSteps",type="integer",JSONPath=".status.stepPhases.RollBacked",description="rollBacked step count"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. "
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// WorkflowList contains a list of Workflow
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
