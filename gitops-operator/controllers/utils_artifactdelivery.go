package controllers

import (
	"context"
	"fmt"
	gitopsv1 "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/api/v1"
	"github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/controllers/common"
	"github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/controllers/common/types"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getArtifactProject(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, "projectmeta.yaml")); err == nil {
		yamlFile, err := ioutil.ReadFile(filepath.Join(dir, "projectmeta.yaml"))
		if err != nil {
			return "", fmt.Errorf("could not read file: %w", err)
		}
		metadataYaml := types.KeptnArtifactRootMetadata{}
		err = yaml.Unmarshal(yamlFile, &metadataYaml)
		if err != nil {
			return "", fmt.Errorf("could not unmarshal file: %w", err)
		}
		return metadataYaml.Spec.Project, nil
	} else {
		return "", fmt.Errorf("could not find a project")
	}
}

func (r *KeptnGitRepositoryReconciler) composeData(gitClient common.GitClient, tmpFs afero.Fs, dir string, services map[types.DirectoryData]types.KeptnArtifactMetadataSpec, stages []types.DirectoryData) error {
	serviceTagExists := false
	// Copy the Base Directory

	for service, metadata := range services {
		if metadata.ConfigVersion == "" {
			metadata.ConfigVersion = "0"
		}
		tag := service.DirectoryName + "-" + metadata.Version + "-" + metadata.ConfigVersion

		err := gitClient.TagExists(tag)
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
		err = gitClient.CommitAndPushUpstream(tag, serviceTagExists)
		if err != nil {
			return fmt.Errorf("could not push to upstream: %w", err)
		}
	}
	return nil
}

func (r *KeptnGitRepositoryReconciler) deliverArtifacts(ctx context.Context, req ctrl.Request, fs afero.Fs, keptnGitRepository *gitopsv1.KeptnGitRepository, codeRepoDir string) error {
	upstreamDir, _ := ioutil.TempDir("", "upstream_tmp_dir")
	artifactBaseRoot := filepath.Join(codeRepoDir, keptnGitRepository.Spec.BaseDir, "base")
	artifactStageRoot := filepath.Join(codeRepoDir, keptnGitRepository.Spec.BaseDir, "stages")
	artifactProject, err := getArtifactProject(artifactBaseRoot)
	if err != nil {
		return err
	}
	if artifactProject != "" {
		upstreamRepo, err := common.GetUpstreamCredentials(ctx, r.Client, artifactProject, req.Namespace)
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

		upstreamGitClient, err := r.GitClientFactory.GetClient(*upstreamRepo, upstreamDir)
		if err != nil {
			return err
		}

		err = r.composeData(upstreamGitClient, fs, upstreamDir, serviceArtifacts, stages)
		if err != nil {
			return err
		}
	}
	return nil
}
