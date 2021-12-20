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

// KeptnShipyardSpec defines the desired state of KeptnShipyard
type KeptnShipyardSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Project string `json:"project" yaml:"project"`

	Stages []KeptnShipyardStage `json:"stages" yaml:"stages,omitempty"`
}

type KeptnShipyardStage struct {
	StageRef string `json:"stageRef" yaml:"stageRef"`
}

// KeptnShipyardStatus defines the observed state of KeptnShipyard
type KeptnShipyardStatus struct {
	ProjectExists   bool   `json:"projectExists,omitempty"`
	LastAppliedHash string `json:"lastAppliedHash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnShipyard is the Schema for the keptnshipyards API
type KeptnShipyard struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnShipyardSpec   `json:"spec,omitempty"`
	Status KeptnShipyardStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnShipyardList contains a list of KeptnShipyard
type KeptnShipyardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnShipyard `json:"items"`
}

type CreateProject struct {

	// git remote URL
	GitRemoteURL string `json:"gitRemoteURL,omitempty"`

	// git token
	GitToken string `json:"gitToken,omitempty"`

	// git user
	GitUser string `json:"gitUser,omitempty"`

	// name
	// Required: true
	Name *string `json:"name"`

	// shipyard
	// Required: true
	Shipyard []byte `json:"shipyard"`
}

func init() {
	SchemeBuilder.Register(&KeptnShipyard{}, &KeptnShipyardList{})
}
