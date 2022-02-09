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

// KeptnInstanceSpec defines the desired state of KeptnInstance
type KeptnInstanceSpec struct {
	APIUrl    string `json:"apiUrl"`
	TokenType string `json:"tokenType,omitempty"`
	Token     string `json:"apiToken,omitempty"`
}

// KeptnInstanceStatus defines the observed state of KeptnInstance
type KeptnInstanceStatus struct {
	AuthHeader   string      `json:"authHeader"`
	CurrentToken string      `json:"currentToken"`
	LastUpdated  metav1.Time `json:"lastUpdated,omitempty"`
	Scheme       string      `json:"APIScheme,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnInstance is the Schema for the keptninstances API
type KeptnInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnInstanceSpec   `json:"spec,omitempty"`
	Status KeptnInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnInstanceList contains a list of KeptnInstance
type KeptnInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnInstance{}, &KeptnInstanceList{})
}
