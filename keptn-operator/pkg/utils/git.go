package utils

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//CheckOutGitRepo Checks out the git repo and returns the commit hash of it
func CheckOutGitRepo(repositoryConfig *gitRepositoryConfig, dir string) (*git.Repository, string, error) {
	authentication := &githttp.BasicAuth{
		Username: repositoryConfig.User,
		Password: repositoryConfig.Token,
	}

	cloneOptions := git.CloneOptions{
		URL:           repositoryConfig.RemoteURI,
		Auth:          authentication,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/main"),
	}

	repo, err := git.PlainClone(dir, false, &cloneOptions)
	if err != nil {
		cloneOptions.ReferenceName = "refs/heads/master"
		repo, err = git.PlainClone(dir, false, &cloneOptions)
		if err != nil {
			return nil, "", fmt.Errorf("Could not checkout "+repositoryConfig.RemoteURI+"/"+repositoryConfig.Branch, err)
		}
	}

	head, err := repo.Head()
	if err != nil {
		return nil, "", fmt.Errorf("Could not get hash of "+repositoryConfig.RemoteURI+"/"+repositoryConfig.Branch, err)
	}
	return repo, head.Hash().String(), nil
}

//GetUpstreamCredentials gets the KeptnProject Resource for the Project and reads the git credentials
func GetUpstreamCredentials(ctx context.Context, client client.Client, project string, namespace string) (*gitRepositoryConfig, error) {
	obj := &apiv1.KeptnProject{}
	err := client.Get(ctx, types.NamespacedName{Name: project, Namespace: namespace}, obj)
	if err != nil {
		return &gitRepositoryConfig{}, err
	}

	return GetGitCredentials(obj.Spec.Repository, obj.Spec.Username, obj.Spec.Password, obj.Spec.DefaultBranch)
}

//GetGitCredentials creates a unified struct for git credentials
func GetGitCredentials(remoteURI, user, token string, branch string) (*gitRepositoryConfig, error) {
	secret, err := DecryptSecret(token)
	if err != nil {
		return nil, err
	}

	if branch == "" {
		branch = "main"
	}

	return &gitRepositoryConfig{
		RemoteURI: remoteURI,
		User:      user,
		Token:     secret,
		Branch:    branch,
	}, nil
}

//AddGit adds a given worktree to git
func AddGit(worktree *git.Worktree) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktree.Filesystem.Root()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("could not add files: %v", err)
	}
	return nil
}
