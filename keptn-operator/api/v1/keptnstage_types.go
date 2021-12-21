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

// KeptnStageSpec defines the desired state of KeptnStage
type KeptnStageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Project defines the Keptn Project this stage is assigned to
	Project string `json:"project"`

	// Sequence defines an array of sequences this KeptnStage will use
	Sequence []KeptnSequenceRefSpec `json:"sequence"`
}

// KeptnStageStatus defines the observed state of KeptnStage
type KeptnStageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type KeptnSequenceRefSpec struct {
	// Type describes how the sequence is defined in this KeptnSequenceRefSpec
	Type string `json:"type"`
	// SequenceRef is used to set a reference to a KeptnSequence
	SequenceRef string `json:"sequenceRef,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnStage is the Schema for the keptnstages API
type KeptnStage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnStageSpec   `json:"spec,omitempty"`
	Status KeptnStageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnStageList contains a list of KeptnStage
type KeptnStageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnStage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnStage{}, &KeptnStageList{})
}
