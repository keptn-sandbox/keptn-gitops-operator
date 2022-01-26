package controllers

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	gitopsv1 "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/api/v1"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getArtifactProject(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, "projectmeta.yaml")); err == nil {
		yamlFile, err := ioutil.ReadFile(filepath.Join(dir, "projectmeta.yaml"))
		if err != nil {
			return "", fmt.Errorf("could not read file: %w", err)
		}
		metadataYaml := KeptnArtifactRootMetadata{}
		err = yaml.Unmarshal(yamlFile, &metadataYaml)
		if err != nil {
			return "", fmt.Errorf("could not unmarshal file: %w", err)
		}
		return metadataYaml.Spec.Project, nil
	} else {
		return "", fmt.Errorf("could not find a project")
	}
}

func composeData(tmpFs afero.Fs, dir string, repo gitRepositoryConfig, gitrepo *git.Repository, services map[DirectoryData]KeptnArtifactMetadataSpec, stages []DirectoryData) error {
	serviceTagExists := false
	// Copy the Base Directory

	for service, metadata := range services {
		tag := service.DirectoryName + "-" + metadata.Version

		err := tagExists(tag, gitrepo)
		if err != nil {
			if err.Error() == "tag was found" && !metadata.OverwriteTag {
				fmt.Println("Would not overwrite Tag")
				continue
			}
			serviceTagExists = true
			fmt.Println("Would overwrite Tag")
		}

		err = cleanupServiceDirs(tmpFs, service.DirectoryName, dir, stages)
		if err != nil {
			return err
		}

		err = CopyDir(tmpFs, service.Path, filepath.Join(dir, "base", service.DirectoryName))
		if err != nil {
			return fmt.Errorf("could not copy "+service.Path+" to tmp: %w", err)
		}

		if metadata.ChartDir != "" {
			err := CopyDir(tmpFs, metadata.ChartDir, filepath.Join(dir, "base", service.DirectoryName, "helm", service.DirectoryName))
			if err != nil {
				return fmt.Errorf("could not copy "+service.Path+" to tmp: %w", err)
			}
		}
		for _, stage := range stages {
			_, err := tmpFs.Stat(filepath.Join(stage.Path, service.DirectoryName))
			if err == nil {
				err := CopyDir(tmpFs, filepath.Join(stage.Path, service.DirectoryName), filepath.Join(dir, "stages", stage.DirectoryName, service.DirectoryName))
				if err != nil {
					return fmt.Errorf("could not copy "+service.Path+" to tmp: %w", err)
				}
				fmt.Println("Stage: "+stage.DirectoryName, "Service: "+service.DirectoryName)
			} else {
				fmt.Println("No Data - Stage: "+stage.DirectoryName, "Service: "+service.DirectoryName)
			}
		}
		err = CommitAndPushUpstream(repo, gitrepo, tag, serviceTagExists)
		if err != nil {
			return fmt.Errorf("could not push to upstream: %w", err)
		}
	}
	return nil
}

func deliverArtifacts(ctx context.Context, req ctrl.Request, client client.Client, fs afero.Fs, keptnGitRepository *gitopsv1.KeptnGitRepository, codeRepoDir string) error {
	upstreamDir, _ := ioutil.TempDir("", "upstream_tmp_dir")
	artifactBaseRoot := filepath.Join(codeRepoDir, keptnGitRepository.Spec.BaseDir, "base")
	artifactStageRoot := filepath.Join(codeRepoDir, keptnGitRepository.Spec.BaseDir, "stages")
	artifactProject, err := getArtifactProject(artifactBaseRoot)
	if err != nil {
		return err
	}
	if artifactProject != "" {
		upstreamRepo, err := getUpstreamCredentials(ctx, client, artifactProject, req.Namespace)
		if err != nil {
			return err
		}

		serviceArtifacts, err := findServiceDirs(fs, filepath.Join(artifactBaseRoot), "metadata.yaml")
		if err != nil {
			return err
		}

		stages, err := findDirs(fs, artifactStageRoot)
		if err != nil {
			return err
		}

		gitrepo, _, err := upstreamRepo.CheckOutGitRepo(upstreamDir)
		if err != nil {
			return err
		}

		err = composeData(fs, upstreamDir, *upstreamRepo, gitrepo, serviceArtifacts, stages)
		if err != nil {
			return err
		}
	}
	return nil
}
