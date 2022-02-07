package common

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	internaltypes "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/controllers/common/types"
	"os/exec"
	"time"
)

type GitClientFactory interface {
	GetClient(repositoryConfig internaltypes.GitRepositoryConfig, dir string) (GitClient, error)
}

type GitClient interface {
	Checkout(config internaltypes.GitRepositoryConfig, directory string) error
	GetLastCommitHash() (string, error)
	TagExists(tag string) error
	CommitAndPushUpstream(tag string, tagExists bool) error
}

type GoGitClientFactory struct{}

func (GoGitClientFactory) GetClient(repositoryConfig internaltypes.GitRepositoryConfig, dir string) (GitClient, error) {
	client := &GoGitClient{}
	if err := client.Checkout(repositoryConfig, dir); err != nil {
		return nil, err
	}
	return client, nil
}

type GoGitClient struct {
	repoConfig internaltypes.GitRepositoryConfig
	repo       *git.Repository
}

// Checkout checks out the given repository and returns the hash of the last commit
func (gc *GoGitClient) Checkout(repositoryConfig internaltypes.GitRepositoryConfig, dir string) error {
	authentication := &githttp.BasicAuth{
		Username: repositoryConfig.User,
		Password: repositoryConfig.Token,
	}

	cloneOptions := git.CloneOptions{
		URL:           repositoryConfig.RemoteURI,
		Auth:          authentication,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + repositoryConfig.Branch),
	}

	repo, err := git.PlainClone(dir, false, &cloneOptions)
	if err != nil {
		return fmt.Errorf("Could not checkout "+repositoryConfig.RemoteURI+"/"+repositoryConfig.Branch, err)
	}

	gc.repo = repo
	gc.repoConfig = repositoryConfig
	return nil
}

func (gc *GoGitClient) TagExists(tag string) error {
	tagFoundErr := "tag was found"
	tags, err := gc.repo.TagObjects()
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

func (gc *GoGitClient) GetLastCommitHash() (string, error) {
	head, err := gc.repo.Head()
	if err != nil {
		return "", fmt.Errorf("Could not get hash of "+gc.repoConfig.RemoteURI+"/"+gc.repoConfig.Branch, err)
	}
	return head.Hash().String(), nil
}

func (gc *GoGitClient) CommitAndPushUpstream(tag string, tagExists bool) error {
	authentication := &githttp.BasicAuth{
		Username: gc.repoConfig.User,
		Password: gc.repoConfig.Token,
	}

	commitOptions := git.CommitOptions{
		Author: &object.Signature{
			Name:  "Keptn Upstream Pusher",
			Email: "keptn@keptn.sh",
			When:  time.Now(),
		},
	}

	w, err := gc.repo.Worktree()
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

	h, err := gc.repo.Head()
	if err != nil {
		return fmt.Errorf("could not get head: %w", err)
	}

	if tagExists {
		err := gc.DeleteTag(w, tag)
		if err != nil {
			return fmt.Errorf("could not delete tag: %w", err)
		}
	}

	if tag != "" {
		_, err = gc.repo.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
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

	err = gc.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       authentication,
		RefSpecs: []config.RefSpec{
			"refs/heads/*:refs/heads/*",
		},
	})
	if err != nil {
		return fmt.Errorf("could not push commit: %w", err)
	}

	err = gc.repo.Push(&git.PushOptions{
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

func (gc *GoGitClient) DeleteTag(worktree *git.Worktree, tag string) error {
	err := gc.repo.DeleteTag(tag)
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
