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

const (
	// KeptnGitRepositoryPhaseSuccessful defines the value for a successful action
	KeptnGitRepositoryPhaseSuccessful = "Successful"
	// KeptnGitRepositoryPhaseFailed defines the value for a failed action
	KeptnGitRepositoryPhaseFailed = "Failed"
)

// KeptnGitRepositorySpec defines the desired state of KeptnGitRepository
type KeptnGitRepositorySpec struct {
	Repository string `json:"repository"`
	Token      string `json:"password"`
	Username   string `json:"username"`
	Branch     string `json:"branch,omitempty"`
	BaseDir    string `json:"baseDir,omitempty"`
}

// KeptnGitRepositoryStatus defines the observed state of KeptnGitRepository
type KeptnGitRepositoryStatus struct {
	LastCommit string `json:"lastCommit,omitempty"`
	Result     string `json:"result,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnGitRepository is the Schema for the keptngitrepositories API
type KeptnGitRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnGitRepositorySpec   `json:"spec,omitempty"`
	Status KeptnGitRepositoryStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnGitRepositoryList contains a list of KeptnGitRepository
type KeptnGitRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnGitRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnGitRepository{}, &KeptnGitRepositoryList{})
}
