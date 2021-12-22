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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
	keptnshv1 "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/api/v1"
	copy "github.com/otiai10/copy"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"os/exec"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

// KeptnGitRepositoryReconciler reconciles a KeptnGitRepository object
type KeptnGitRepositoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

type gitCredentials struct {
	User      string `json:"user,omitempty"`
	Token     string `json:"token,omitempty"`
	RemoteURI string `json:"remoteURI"`
}

type deployment struct {
	Metadata metadata `yaml:"metadata"`
}

type metadata struct {
	ImageVersion string `yaml:"imageVersion"`
}

//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories/finalizers,verbs=update
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequences/,verbs=get;list
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects/,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnGitRepository object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnGitRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Reconciling KeptnGitRepository")

	keptnGitRepository := &keptnshv1.KeptnGitRepository{}
	err := r.Get(ctx, req.NamespacedName, keptnGitRepository)
	if errors.IsNotFound(err) {
		r.Log.Info("KeptnGitRepository resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	r.Log.Info("Syncing", "url", keptnGitRepository.Spec.Repository)
	r.Log.Info("Syncing", "status", keptnGitRepository.Status)

	credentials, err := r.getGitCredentials(keptnGitRepository)
	if err != nil {
		r.updateStatusResult(ctx, keptnGitRepository, keptnshv1.KeptnGitRepositoryPhaseFailed)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	dir, err := cloneGitRepository(keptnGitRepository.Spec.Repository, credentials)
	if err != nil {
		r.Log.Error(err, "Could not clone", "URI", keptnGitRepository.Spec.Repository)
		r.updateStatusResult(ctx, keptnGitRepository, keptnshv1.KeptnGitRepositoryPhaseFailed)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	keptnProject, err := r.applyKubernetesResources(dir, req.Namespace)
	if err != nil {
		r.Log.Error(err, "Could not apply kubernetes resources")
		r.updateStatusResult(ctx, keptnGitRepository, keptnshv1.KeptnGitRepositoryPhaseFailed)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	keptnProjectCredentials, err := r.getGitCredentialsProject(keptnProject)
	if err != nil {
		r.Log.Error(err, "Could not get secret for project"+keptnProject.Name)
		r.updateStatusResult(ctx, keptnGitRepository, keptnshv1.KeptnGitRepositoryPhaseFailed)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if keptnProject.Spec.DefaultBranch == "" {
		keptnProject.Spec.DefaultBranch = "master"
	}

	err = r.pushKeptnResources(dir, keptnProjectCredentials, keptnProject.Spec.DefaultBranch)
	if err != nil {
		r.Log.Error(err, "Could not push project resources to keptn git repository")
		r.updateStatusResult(ctx, keptnGitRepository, keptnshv1.KeptnGitRepositoryPhaseFailed)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	r.Log.Info("Finished Reconciling")
	r.updateStatusResult(ctx, keptnGitRepository, keptnshv1.KeptnGitRepositoryPhaseSuccessful)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *KeptnGitRepositoryReconciler) updateStatusResult(ctx context.Context, keptnGitRepository *keptnshv1.KeptnGitRepository, result string) {
	if keptnGitRepository.Status.Result != result {
		keptnGitRepository.Status.Result = result
		err := r.Client.Status().Update(ctx, keptnGitRepository)
		if err != nil {
			r.Log.Error(err, "Could not update status", "keptnGitRepository", keptnGitRepository.Name)
		} else {
			r.Log.Info("Updated status", "status", keptnGitRepository.Status)
		}
	}
}

func (r *KeptnGitRepositoryReconciler) getGitCredentialsProject(project *keptnshv1.KeptnProject) (*gitCredentials, error) {
	secret, err := decryptSecret(project.Spec.Password)
	if err != nil {
		fmt.Println("could not decrypt secret")
		return &gitCredentials{}, err
	}

	credentials := &gitCredentials{}
	credentials.Token = secret
	credentials.RemoteURI = project.Spec.Repository
	credentials.User = project.Spec.Username
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal secret: %w", err)
	}

	return credentials, nil
}

func (r *KeptnGitRepositoryReconciler) getGitCredentials(gitrepo *keptnshv1.KeptnGitRepository) (*gitCredentials, error) {
	secret, err := decryptSecret(gitrepo.Spec.Token)
	if err != nil {
		fmt.Println("could not decrypt secret")
		return &gitCredentials{}, err
	}

	credentials := &gitCredentials{}
	credentials.Token = secret
	credentials.RemoteURI = gitrepo.Spec.Repository
	credentials.User = gitrepo.Spec.Username
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal secret: %w", err)
	}

	return credentials, nil
}

func (r *KeptnGitRepositoryReconciler) getGitCredentialsFromSecret(ctx context.Context, secretName string, namespace string) (*gitCredentials, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve secret: %w", err)
	}

	credentials := &gitCredentials{}
	err = json.Unmarshal(secret.Data["git-credentials"], &credentials)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal secret: %w", err)
	}

	return credentials, nil
}

func cloneGitRepository(url string, credentials *gitCredentials) (string, error) {
	dir, _ := ioutil.TempDir("", "tmp")
	_, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: url,
		Auth: &http.BasicAuth{
			Username: credentials.User,
			Password: credentials.Token,
		},
		SingleBranch: true,
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (r *KeptnGitRepositoryReconciler) applyKubernetesResources(dir string, namespace string) (*keptnshv1.KeptnProject, error) {
	keptnProject := &keptnshv1.KeptnProject{}
	keptnPath := filepath.Join(dir, ".keptn")

	configFiles := []string{"stage.yaml", "service.yaml", "sequences.yaml", "project.yaml", "task.yaml"}

	for _, configFile := range configFiles {
		configPath := filepath.Join(keptnPath, configFile)
		if _, err := os.Stat(configPath); err == nil {
			r.Log.Info("Apply kubernetes resource", "configFile", configFile)
			output, err := exec.Command("kubectl", "apply", "-n", namespace, "-f", configPath).CombinedOutput()
			if err != nil {
				return keptnProject, fmt.Errorf("%w: kubectl output %s", err, output)
			}

			if configFile == "project.yaml" {
				err = unmarshalYAMLFile(configPath, keptnProject)
				if err != nil {
					return keptnProject, err
				}
			}
		} else {
			r.Log.Info("Not found, skipping", "configFile", configFile)
		}
	}

	return keptnProject, nil
}

func unmarshalYAMLFile(file string, value interface{}) error {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", file, err)
	}

	err = yaml.Unmarshal(yamlFile, value)
	if err != nil {
		return fmt.Errorf("could not unmarshal: %w: %s", err, yamlFile)
	}
	return nil
}

func (r *KeptnGitRepositoryReconciler) pushKeptnResources(sourceDir string, gitCredentials *gitCredentials, branch string) error {
	authentication := &http.BasicAuth{
		Username: gitCredentials.User,
		Password: gitCredentials.Token,
	}

	destinationDir, _ := ioutil.TempDir("", "tmp")
	repository, err := git.PlainClone(destinationDir, false, &git.CloneOptions{
		URL:           gitCredentials.RemoteURI,
		Auth:          authentication,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
	})
	if err != nil {
		return fmt.Errorf("could not clone %s: %w", destinationDir, err)
	}

	sourcePath := filepath.Join(sourceDir, ".keptn/base")
	destinationPath := filepath.Join(destinationDir, "base")
	err = copy.Copy(sourcePath, destinationPath)
	if err != nil {
		return fmt.Errorf("could not copy src %s to %s: %w", sourcePath, destinationPath, err)
	}

	sourcePath = filepath.Join(sourceDir, ".keptn/stages")
	destinationPath = filepath.Join(destinationDir, "stages")
	err = copy.Copy(sourcePath, destinationPath)
	if err != nil {
		return fmt.Errorf("could not copy src %s to %s: %w", sourcePath, destinationPath, err)
	}

	w, err := repository.Worktree()
	if err != nil {
		return fmt.Errorf("could not set worktree: %w", err)
	}

	signature := &object.Signature{
		Name:  "keptn-git-operator",
		Email: "keptn-git-operator@keptn.sh",
		When:  time.Now(),
	}

	commitOptions := git.CommitOptions{
		Author: signature,
	}

	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("could not add files: %w", err)
	}

	tag, err := getImageVersionFromConfiguration(destinationDir)
	if err != nil {
		return fmt.Errorf("could not get imageVersion: %w", err)
	}
	r.Log.Info("tag", "tag", tag)

	_, err = w.Commit("Add resources of version: "+tag, &commitOptions)
	if err != nil {
		return fmt.Errorf("could not commit: %w", err)
	}

	h, err := repository.Head()

	repository.CreateTag(tag, h.Hash(), &git.CreateTagOptions{
		Tagger:  signature,
		Message: tag,
	})
	if err != nil {
		return fmt.Errorf("couldn't create tag: %w", err)
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
		r.Log.Error(err, "Couldn't push tag, probably already exists or no changes, ignoring", "tag", tag)
	}

	return nil
}

func getImageVersionFromConfiguration(dir string) (string, error) {
	deployment := &deployment{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("could not walk filepath: %w", err)
		}

		if strings.HasSuffix(path, filepath.Join("metadata", "deployment.yaml")) {
			err = unmarshalYAMLFile(path, deployment)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return deployment.Metadata.ImageVersion, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnGitRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keptnshv1.KeptnGitRepository{}).
		Complete(r)
}

// For the solution without kubectl apply ;)
// https://github.com/kubernetes/client-go/issues/193#issuecomment-363318588
/*func parseK8sYaml(fileR []byte) []runtime.Object {
	acceptedK8sTypes := regexp.MustCompile(`(KeptnGitRepository)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]runtime.Object, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(f), nil, nil)

		if err != nil {
			log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			continue
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			log.Printf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			retVal = append(retVal, obj)
		}
	}
	return retVal
}*/
