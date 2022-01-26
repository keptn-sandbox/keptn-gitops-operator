package controllers

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func tagExists(tag string, r *git.Repository) error {
	tagFoundErr := "tag was found"
	tags, err := r.TagObjects()
	if err != nil {
		return fmt.Errorf("get tags error: %w", err)
	}

	err = tags.ForEach(func(t *object.Tag) error {
		if t.Name == tag {
			return fmt.Errorf(tagFoundErr)
		}
		return nil
	})
	return err
}

func DeleteGitTag(worktree *git.Worktree, repository *git.Repository, tag string) error {
	err := repository.DeleteTag(tag)
	if err != nil {
		return fmt.Errorf("could not remove local tag: %w", err)
	}
	cmd := exec.Command("git", "push", "--delete", "origin", tag)
	cmd.Dir = worktree.Filesystem.Root()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("could not delete tag: %w", err)
	}
	return nil
}

func AddGit(worktree *git.Worktree) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = worktree.Filesystem.Root()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Could not add files: %v", err)
	}
	return nil
}

func CommitAndPushUpstream(repoconfig gitRepositoryConfig, repository *git.Repository, tag string, tagexists bool) error {
	authentication := &githttp.BasicAuth{
		Username: repoconfig.user,
		Password: repoconfig.token,
	}

	commitOptions := git.CommitOptions{
		Author: &object.Signature{
			Name:  "Keptn Upstream Pusher",
			Email: "keptn@keptn.sh",
			When:  time.Now(),
		},
	}

	w, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("could not set worktree: %w", err)
	}

	// go-git can't stage deleted files https://github.com/src-d/go-git/issues/1268
	err = AddGit(w)
	if err != nil {
		return fmt.Errorf("could not add files: %w", err)
	}

	_, err = w.Commit("Push new version", &commitOptions)
	if err != nil {
		return fmt.Errorf("could not commit: %w", err)
	}

	h, err := repository.Head()
	if err != nil {
		return fmt.Errorf("could not get head: %w", err)
	}

	if tagexists {
		err := DeleteGitTag(w, repository, tag)
		if err != nil {
			return fmt.Errorf("could not delete tag: %w", err)
		}
	}

	if tag != "" {
		_, err = repository.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
			Tagger: &object.Signature{
				Name:  "Keptn Upstream Pusher",
				Email: "keptn@keptn.sh",
				When:  time.Now(),
			},
			Message: "Created a Tag",
		})
		if err != nil {
			return fmt.Errorf("couldn't create a tag: %w", err)
		}
	}

	err = repository.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       authentication,
		RefSpecs: []config.RefSpec{
			"refs/heads/*:refs/heads/*",
		},
	})
	if err != nil {
		return fmt.Errorf("could not push commit: %w", err)
	}

	err = repository.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       authentication,
		RefSpecs: []config.RefSpec{
			"refs/tags/*:refs/tags/*",
		},
	})
	if err != nil {
		return fmt.Errorf("could not push tags: %w", err)
	}

	return nil
}

func (repositoryConfig *gitRepositoryConfig) CheckOutGitRepo(dir string) (*git.Repository, string, error) {
	authentication := &githttp.BasicAuth{
		Username: repositoryConfig.user,
		Password: repositoryConfig.token,
	}

	cloneOptions := git.CloneOptions{
		URL:           repositoryConfig.remoteURI,
		Auth:          authentication,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + repositoryConfig.branch),
	}

	repo, err := git.PlainClone(dir, false, &cloneOptions)
	if err != nil {
		return nil, "", fmt.Errorf("Could not checkout "+repositoryConfig.remoteURI+"/"+repositoryConfig.branch, err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, "", fmt.Errorf("Could not get hash of "+repositoryConfig.remoteURI+"/"+repositoryConfig.branch, err)
	}
	return repo, head.Hash().String(), nil
}

func getUpstreamCredentials(ctx context.Context, client client.Client, project string, namespace string) (*gitRepositoryConfig, error) {
	obj := &keptnv1.KeptnProject{}
	err := client.Get(ctx, types.NamespacedName{Name: project, Namespace: namespace}, obj)
	if err != nil {
		return &gitRepositoryConfig{}, err
	}

	return getGitCredentials(obj.Spec.Repository, obj.Spec.Username, obj.Spec.Password, obj.Spec.DefaultBranch)
}

func getGitCredentials(remoteURI, user, token string, branch string) (*gitRepositoryConfig, error) {
	secret, err := decryptSecret(token)
	if err != nil {
		return nil, err
	}

	if branch == "" {
		branch = "main"
	}

	return &gitRepositoryConfig{
		remoteURI: remoteURI,
		user:      user,
		token:     secret,
		branch:    branch,
	}, nil
}
