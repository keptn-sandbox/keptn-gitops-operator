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

// KeptnServiceDeploymentSpec defines the desired state of KeptnServiceDeployment
type KeptnServiceDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Project          string `json:"project"`
	Service          string `json:"service"`
	Stage            string `json:"stage"`
	Version          string `json:"version"`
	ConfigVersion    string `json:"configVersion,omitempty"`
	Author           string `json:"author,omitempty"`
	SourceCommitHash string `json:"sourceCommitHash,omitempty"`
}

// KeptnServiceDeploymentStatus defines the observed state of KeptnServiceDeployment
type KeptnServiceDeploymentStatus struct {
	DeployedVersion       string                              `json:"deployedVersion,omitempty"`
	DeployedConfigVersion string                              `json:"deployedConfigVersion,omitempty"`
	UpdatePending         bool                                `json:"updatePending,omitempty"`
	KeptnContext          string                              `json:"keptnContext,omitempty"`
	LastAppliedHash       string                              `json:"lastAppliedHash,omitempty"`
	Prerequisites         KeptnServiceDeploymentPrerequisites `json:"prerequisites,omitempty"`
	DeploymentProgress    KeptnServiceDeploymentProgress      `json:"progress,omitempty"`
}

//KeptnServiceDeploymentPrerequisites defines all of the objects needed to deploy a service
type KeptnServiceDeploymentPrerequisites struct {
	ProjectExists bool `json:"projectExists,omitempty"`
	ServiceExists bool `json:"serviceExists,omitempty"`
	StageExists   bool `json:"stageExists,omitempty"`
}

//KeptnServiceDeploymentProgress describes the state of the deployment progress
type KeptnServiceDeploymentProgress struct {
	ArtifactAvailable   bool `json:"artifactAvailable,omitempty"`
	DeploymentTriggered bool `json:"deploymentTriggered,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KeptnServiceDeployment is the Schema for the keptnservicedeployments API
type KeptnServiceDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeptnServiceDeploymentSpec   `json:"spec,omitempty"`
	Status KeptnServiceDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KeptnServiceDeploymentList contains a list of KeptnServiceDeployment
type KeptnServiceDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeptnServiceDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeptnServiceDeployment{}, &KeptnServiceDeploymentList{})
}
