package common

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	commontypes "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/controllers/common/types"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//AddGit adds changes to a git repositoray
func AddGit(worktree *git.Worktree) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktree.Filesystem.Root()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Could not add files: %v", err)
	}
	return nil
}

//GetUpstreamCredentials fetches the credentials for the Kpetn upstream from a KeptnProject CRD
func GetUpstreamCredentials(ctx context.Context, client client.Client, project string, namespace string) (*commontypes.GitRepositoryConfig, error) {
	obj := &keptnv1.KeptnProject{}
	err := client.Get(ctx, types.NamespacedName{Name: project, Namespace: namespace}, obj)
	if err != nil {
		return &commontypes.GitRepositoryConfig{}, err
	}

	return GetGitCredentials(obj.Spec.Repository, obj.Spec.Username, obj.Spec.Password, obj.Spec.DefaultBranch)
}

//GetGitCredentials creates git credentials struct from information
func GetGitCredentials(remoteURI, user, token string, branch string) (*commontypes.GitRepositoryConfig, error) {
	secret, err := decryptSecret(token)
	if err != nil {
		return nil, err
	}

	if branch == "" {
		branch = "main"
	}

	return &commontypes.GitRepositoryConfig{
		RemoteURI: remoteURI,
		User:      user,
		Token:     secret,
		Branch:    branch,
	}, nil
}
