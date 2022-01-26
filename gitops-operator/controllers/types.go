package controllers

import (
	"github.com/go-git/go-git/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DirectoryData struct {
	DirectoryName string
	Path          string
}

type KeptnArtifactRootMetadata struct {
	TypeMeta metav1.TypeMeta               `json:"inline"`
	Spec     KeptnArtifactRootMetadataSpec `json:"spec"`
}

type KeptnArtifactRootMetadataSpec struct {
	Project string `json:"string"`
}

type KeptnArtifactMetadata struct {
	TypeMeta metav1.TypeMeta           `json:"inline"`
	Spec     KeptnArtifactMetadataSpec `json:"spec"`
}

type KeptnArtifactMetadataSpec struct {
	Version      string `json:"version,omitempty"`
	ChartDir     string `json:"chartDir,omitempty"`
	OverwriteTag bool   `json:"overwriteTag,omitempty"`
	Project      string `json:"project"`
}

type configurationData struct {
	credentials *gitRepositoryConfig
	repo        *git.Repository
	tmpDir      string
	stages      []DirectoryData
	services    map[DirectoryData]KeptnArtifactMetadataSpec
}

type gitRepositoryConfig struct {
	remoteURI string
	user      string
	token     string
	branch    string
}
