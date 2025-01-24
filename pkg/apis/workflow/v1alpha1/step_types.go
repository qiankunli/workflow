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

type RetryPolicy struct { // 描述该step 依赖其他step的情况
	// +kubebuilder:default:=3
	RunRetryLimit int32 `json:"runRetryLimit,omitempty"`
	// +kubebuilder:default:=60
	RunRetryPeriodSeconds int32 `json:"runRetryPeriodSeconds,omitempty"`
	// +kubebuilder:default:=3
	RollbackRetryLimit int32 `json:"rollbackRetryLimit,omitempty"`
	// +kubebuilder:default:=60
	RollbackRetryPeriodSeconds int32 `json:"rollbackRetryPeriodSeconds,omitempty"`
}

// StepSpec defines the desired state of Step
type StepSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Type string `json:"type,omitempty"`
	// Json类型的数据
	Data string `json:"data,omitempty"`
	// Map类型的数据
	Parameters map[string]string `json:"parameters,omitempty"`
	// +kubebuilder:default:=PreserveOnFailure
	RollbackPolicy RollbackPolicy `json:"rollbackPolicy,omitempty"`
	RetryPolicy    RetryPolicy    `json:"retryPolicy,omitempty"`
}

// StepPhase
// +kubebuilder:validation:Enum=Pending;Running;Success;RollingBack;RollBacked;Failed
type StepPhase string

const (
	StepPending     StepPhase = "Pending"
	StepRunning     StepPhase = "Running"
	StepSuccess     StepPhase = "Success"
	StepRollingBack StepPhase = "RollingBack"
	StepRollBacked  StepPhase = "RollBacked"
	StepFailed      StepPhase = "Failed"
)

type StepResource struct {
	Status     string            `json:"status,omitempty"`
	ID         string            `json:"id,omitempty"`
	Name       string            `json:"Name,omitempty"`
	Address    string            `json:"address,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// StepStatus defines the observed state of Step
type StepStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default:=Pending
	Phase StepPhase `json:"phase,omitempty"`

	Resource StepResource `json:"resource,omitempty"`
	// 这里的attributes 将会被合入到workflow 的attributes 中，通过workflow.attributes在多step间传递数据
	Attributes            map[string]string `json:"attributes,omitempty"`
	RunRetryCount         int32             `json:"runRetryCount,omitempty"`
	LatestRunRetryAt      metav1.Time       `json:"latestRunRetryAt,omitempty"`
	RollbackRetryCount    int32             `json:"rollbackRetryCount,omitempty"`
	LatestRollbackRetryAt metav1.Time       `json:"latestRollbackRetryAt,omitempty"`
	RunError              string            `json:"runError,omitempty"`
	RollbackError         string            `json:"rollbackError,omitempty"`
}

// Step is the Schema for the steps API
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=steps,scope=Namespaced
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="step type."
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="step phase. "
// +kubebuilder:printcolumn:name="Address",type="string",JSONPath=".status.resource.address",description="step resource address. "
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.resource.status",description="step resource errorCode. "
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. "
type Step struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StepSpec   `json:"spec,omitempty"`
	Status StepStatus `json:"status,omitempty"`
}

// StepList contains a list of Step
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StepList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Step `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Step{}, &StepList{})
}

const (
	PhaseChangeReason   = "PhaseChange"
	FailedOrErrorReason = "FailedOrError"
	SpecWrongReason     = "SpecWrong"
)
