/*
Copyright 2021.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeptnSequenceExecutionSpec defines the desired state of KeptnSequenceExecution
type KeptnSequenceExecutionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KeptnSequenceExecution. Edit keptnsequenceexecution_types.go to remove/update
	Project string            `json:"project"`
	Service string            `json:"service"`
	Stage   string            `json:"stage"`
	Event   string            `json:"event"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// KeptnSequenceExecutionStatus defines the observed state of KeptnSequenceExecution
type KeptnSequenceExecutionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ProjectExists   bool   `json:"projectExists,omitempty"`
	ServiceExists   bool   `json:"serviceExists,omitempty"`
	KeptnContext    string `json:"keptnContext,omitempty"`
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`
	UpdatePending   bool   `json:"updatePending,omitempty"`
}

//+kubebuilder:resource:shortName=kse
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnSequenceExecution is the Schema for the keptnsequenceexecutions API
type KeptnSequenceExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnSequenceExecutionSpec   `json:"spec,omitempty"`
	Status KeptnSequenceExecutionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnSequenceExecutionList contains a list of KeptnSequenceExecution
type KeptnSequenceExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnSequenceExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnSequenceExecution{}, &KeptnSequenceExecutionList{})
}
