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

// KeptnProjectSpec defines the desired state of KeptnProject
type KeptnProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of KeptnProject. Edit KeptnProject_types.go to remove/update
	Project string `json:"project,omitempty"`
}

// KeptnProjectStatus defines the observed state of KeptnProject
type KeptnProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastMainCommit   string `json:"mainCommit,omitempty"`
	LastDeployCommit string `json:"deployCommit,omitempty"`
	WatchedBranch    string `json:"watchedBranch,omitempty"`
}

// +kubebuilder:object:root=true

// KeptnProject is the Schema for the keptnprojects API
type KeptnProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnProjectSpec   `json:"spec,omitempty"`
	Status KeptnProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeptnProjectList contains a list of KeptnProject
type KeptnProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnProject{}, &KeptnProjectList{})
}
