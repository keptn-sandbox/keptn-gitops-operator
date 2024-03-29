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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptndeploymentcontexts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptndeploymentcontexts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptndeploymentcontexts/finalizers,verbs=update

// KeptnDeploymentContextSpec defines the desired state of KeptnDeploymentContext
type KeptnDeploymentContextSpec struct {
	Project string `json:"project"`
	Service string `json:"service"`
	Version string `json:"version"`
}

// KeptnDeploymentContextStatus defines the observed state of KeptnDeploymentContext
type KeptnDeploymentContextStatus struct {
	LastAppliedHash map[string]string `json:"lastAppliedHash,omitempty"`
	KeptnContext    string            `json:"keptnContext"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnDeploymentContext is the Schema for the keptndeploymentcontexts API
type KeptnDeploymentContext struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnDeploymentContextSpec   `json:"spec,omitempty"`
	Status KeptnDeploymentContextStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnDeploymentContextList contains a list of KeptnDeploymentContext
type KeptnDeploymentContextList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnDeploymentContext `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnDeploymentContext{}, &KeptnDeploymentContextList{})
}
