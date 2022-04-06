package types

import (
	"github.com/go-git/go-git/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//DirectoryData contains information about the service and directory
type DirectoryData struct {
	DirectoryName string
	Path          string
}

//KeptnArtifactRootMetadata is used to get information about the project of metadata
type KeptnArtifactRootMetadata struct {
	TypeMeta metav1.TypeMeta               `json:"inline"`
	Spec     KeptnArtifactRootMetadataSpec `json:"spec"`
}

//KeptnArtifactRootMetadataSpec specifies the artifact metadata on a project level
type KeptnArtifactRootMetadataSpec struct {
	Project string `json:"string"`
}

//KeptnArtifactMetadata provides metadata of service artifacts
type KeptnArtifactMetadata struct {
	TypeMeta metav1.TypeMeta           `json:"inline"`
	Spec     KeptnArtifactMetadataSpec `json:"spec"`
}

//KeptnArtifactMetadataSpec specifies details of service artifacts
type KeptnArtifactMetadataSpec struct {
	Version       string `json:"version,omitempty"`
	ConfigVersion string `json:"configVersion,omitempty"`
	ChartDir      string `json:"chartDir,omitempty"`
	OverwriteTag  bool   `json:"overwriteTag,omitempty"`
	Project       string `json:"project"`
}

type configurationData struct {
	credentials *GitRepositoryConfig
	repo        *git.Repository
	tmpDir      string
	stages      []DirectoryData
	services    map[DirectoryData]KeptnArtifactMetadataSpec
}

//GitRepositoryConfig contains information to interact with a git repo
type GitRepositoryConfig struct {
	RemoteURI string
	User      string
	Token     string
	Branch    string
}
