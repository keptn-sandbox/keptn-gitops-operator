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

// KeptnSequenceSpec defines the desired state of KeptnSequence
type KeptnSequenceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this filee
	// Foo is an example field of KeptnSequence. Edit keptnsequence_types.go to remove/update
	Sequence Sequence `json:"sequence"`
}

// KeptnSequenceStatus defines the observed state of KeptnSequence
type KeptnSequenceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnSequence is the Schema for the keptnsequences API
type KeptnSequence struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnSequenceSpec   `json:"spec,omitempty"`
	Status KeptnSequenceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnSequenceList contains a list of KeptnSequence
type KeptnSequenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnSequence `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnSequence{}, &KeptnSequenceList{})
}

// Sequence defines a task sequence by its name and tasks. The triggers property is optional
type Sequence struct {
	Name        string    `json:"name" yaml:"name"`
	TriggeredOn []Trigger `json:"triggeredOn,omitempty" yaml:"triggeredOn,omitempty"`
	Tasks       []Task    `json:"tasks" yaml:"tasks"`
}

// Task defines a task by its name and optional properties
type Task struct {
	Name           string            `json:"name" yaml:"name"`
	TriggeredAfter string            `json:"triggeredAfter,omitempty" yaml:"triggeredAfter,omitempty"`
	Properties     map[string]string `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// Trigger defines a trigger which causes a sequence to get activated
type Trigger struct {
	Event    string   `json:"event" yaml:"event"`
	Selector Selector `json:"selector,omitempty" yaml:"selector,omitempty"`
}

type Selector struct {
	Match map[string]string `json:"match" yaml:"match"`
}
