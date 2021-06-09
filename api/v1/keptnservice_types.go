/*


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

//+kubebuilder:resource:scope=Namespace

// KeptnServiceSpec defines the desired state of KeptnService
type KeptnServiceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KeptnService. Edit KeptnService_types.go to remove/update
	Project        string `json:"project,omitempty"`
	Service        string `json:"service,omitempty"`
	TriggerCommand string `json:"trigger,omitempty"`
	StartStage     string `json:"startstage,omitempty"`
}

// KeptnServiceStatus defines the observed state of KeptnService
type KeptnServiceStatus struct {
	LastDeployed         string `json:"lastdeployed,omitempty"`
	LastSetupStatus      int    `json:"lastsetupstate,omitempty"`
	DeploymentPending    bool   `json:"deloymentpending,omitempty"`
	DeletionPending      bool   `json:"deletionpending,omitempty"`
	SafeToDelete         bool   `json:"safetodelete,omitempty"`
	DesiredVersion       string `json:"desiredversion,omitempty"`
	CreationPending      bool   `json:"creationpending,omitempty"`
	LastAuthor           string `json:"author,omitempty"`
	LastSourceCommitHash string `json:"sourceCommitHash,omitempty"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// KeptnService is the Schema for the keptnservices API
type KeptnService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnServiceSpec   `json:"spec,omitempty"`
	Status KeptnServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeptnServiceList contains a list of KeptnService
type KeptnServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnService{}, &KeptnServiceList{})
}
