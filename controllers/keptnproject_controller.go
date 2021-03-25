/*


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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	hashdir "github.com/sger/go-hashdir"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keptnv1 "keptn-operator/api/v1"
)

type GitCredentials struct {
	User      string `json:"user,omitempty"`
	Token     string `json:"token,omitempty"`
	RemoteURI string `json:"remoteURI,omitempty"`
}

type KeptnConfig struct {
	Metadata KeptnConfigMeta `yaml:"metadata,omitempty"`
	Services []KeptnService  `yaml:"services,omitempty"`
}

type KeptnConfigMeta struct {
	Branch string `yaml:"initbranch,omitempty"`
}

type KeptnService struct {
	Name              string `yaml:"name,omitempty"`
	DeploymentTrigger string `yaml:"triggerevent"`
}

type KeptnTriggerEvent struct {
	ContentType string         `json:"contenttype,omitempty"`
	Data        KeptnEventData `json:"data,omitempty"`
	Source      string         `json:"source,omitempty"`
	SpecVersion string         `json:"specversion,omitempty"`
	Type        string         `json:"type,omitempty"`
}

type KeptnEventData struct {
	Project string `json:"project,omitempty"`
	Service string `json:"service,omitempty"`
	Stage   string `json:"stage,omitempty"`
}

// KeptnProjectReconciler reconciles a KeptnProject object
type KeptnProjectReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	ReqLogger logr.Logger
}

// +kubebuilder:rbac:groups=keptn.operator.keptn.sh,resources=keptnprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keptn.operator.keptn.sh,resources=keptnprojects/status,verbs=get;update;patch

func (r *KeptnProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	r.ReqLogger = r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)

	r.ReqLogger.Info("Reconciling KeptnProject")

	project := &keptnv1.KeptnProject{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, project)
	if errors.IsNotFound(err) {
		r.ReqLogger.Info("KeptnProject resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	secret := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: "git-credentials-" + project.Name, Namespace: req.Namespace}, secret)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get secret for project "+project.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	var credentials GitCredentials
	err = json.Unmarshal(secret.Data["git-credentials"], &credentials)
	if err != nil {
		r.ReqLogger.Error(err, "Could not unmarshal credentials for project "+project.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	mainHead, err := r.getCommitHash(credentials, "")
	if err != nil {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	config := &KeptnConfig{}

	// Save new git hashes, if changed

	// GET Configuration
	dir, _ := ioutil.TempDir("", "temp_dir")

	_, err = git.PlainClone(dir, false, &git.CloneOptions{
		URL: credentials.RemoteURI,
		Auth: &http.BasicAuth{
			Username: credentials.User,
			Password: credentials.Token,
		},
		SingleBranch: true,
	})
	if err != nil {
		r.ReqLogger.Error(err, "Could not checkout "+project.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if _, err := os.Stat(filepath.Join(dir, ".keptn/config.yaml")); err == nil {
		yamlFile, err := ioutil.ReadFile(filepath.Join(dir, ".keptn/config.yaml"))
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		err = yaml.Unmarshal(yamlFile, config)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		project.Status.WatchedBranch = config.Metadata.Branch

		for _, service := range config.Services {
			err = r.createKeptnService(project, service, req.Namespace)
			if err != nil {
				r.ReqLogger.Error(err, "Could not create service "+project.Name+"/"+service.Name)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
		}
	} else {
		r.ReqLogger.Info("There is no configuration file for project " + project.Name)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	defer os.RemoveAll(dir)

	for _, service := range r.getKeptnServices(project.Name).Items {
		found := false
		for _, configService := range config.Services {
			if service.Spec.Project == project.Name && service.Spec.Service == configService.Name {
				found = true
			}
		}
		if found == false {
			err = r.removeService(project.Name, service.Spec.Service, req.Namespace)
			if err != nil {
				r.ReqLogger.Error(err, "Could not remove Service "+service.Spec.Service)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, err
			}
			return ctrl.Result{}, nil
		}
	}

	project.Status.LastMainCommit = mainHead

	if project.Status.WatchedBranch != "" {
		appCommitHash, err := r.getCommitHash(credentials, project.Status.WatchedBranch)
		if err != nil {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}

		if project.Status.LastDeployCommit != appCommitHash {
			r.ReqLogger.Info("App Branch State has changed - Triggering new Deployment")
			for _, service := range config.Services {
				hash, err := r.getServiceHash(credentials, config.Metadata.Branch, service)
				if err != nil {
					r.ReqLogger.Error(err, "Could not get Hash for Service "+service.Name)
					return ctrl.Result{RequeueAfter: 30 * time.Second}, err
				}
				err = r.triggerDeployment(project.Name, service, config.Metadata.Branch, req.Namespace, hash)
				if err != nil {
					r.ReqLogger.Error(err, "Could not trigger deployment "+service.Name)
					return ctrl.Result{RequeueAfter: 30 * time.Second}, err
				}
			}
			project.Status.LastDeployCommit = appCommitHash
		}
	}

	err = r.Client.Update(context.TODO(), project)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update LastAppCommit")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	r.ReqLogger.Info("Finished Reconciling")

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *KeptnProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keptnv1.KeptnProject{}).
		Complete(r)
}

func (r *KeptnProjectReconciler) createKeptnService(project *keptnv1.KeptnProject, service KeptnService, namespace string) error {
	currentKService := keptnv1.KeptnService{}
	kService := keptnv1.KeptnService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      project.Name + "-" + service.Name,
			Labels: map[string]string{
				"project": project.Name,
			},
		},
		Spec: keptnv1.KeptnServiceSpec{
			Project:        project.Name,
			Service:        service.Name,
			TriggerCommand: service.DeploymentTrigger,
		},
		Status: keptnv1.KeptnServiceStatus{
			CreationPending: true,
		},
	}

	if err := controllerutil.SetControllerReference(project, &kService, r.Scheme); err != nil {
		r.ReqLogger.Error(err, "Failed setting Controller Reference for Service"+service.Name)
		return err
	}

	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: project.Name + "-" + service.Name, Namespace: namespace}, &currentKService); err != nil && errors.IsNotFound(err) {
		log.Println("Creating a new " + service.Name + "Service")
		err = r.Client.Create(context.TODO(), &kService)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *KeptnProjectReconciler) triggerDeployment(project string, service KeptnService, stage string, namespace string, hash string) error {

	keptnService := keptnv1.KeptnService{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: project + "-" + service.Name, Namespace: namespace}, &keptnService)

	if hash != keptnService.Status.Hash {
		keptnService.Status.DeploymentPending = true
		keptnService.Spec.StartStage = stage
		keptnService.Status.Hash = hash

		err = r.Client.Update(context.TODO(), &keptnService)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update KeptnService "+service.Name)
			return err
		} else {
			r.ReqLogger.Info("Updated Service")
		}
	}
	return nil
}

func (r *KeptnProjectReconciler) removeService(project string, service string, namespace string) error {

	keptnService := keptnv1.KeptnService{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: project + "-" + service, Namespace: namespace}, &keptnService)

	if keptnService.Status.SafeToDelete == true {
		err = r.Client.Delete(context.TODO(), &keptnService)
		if err != nil {
			r.ReqLogger.Error(err, "Deletion of "+keptnService.Name+" was unsuccessful")
			return err
		} else {
			r.ReqLogger.Info("Deletion of " + keptnService.Name + " was successful")
			return nil
		}
	}

	keptnService.Status.DeletionPending = true
	err = r.Client.Update(context.TODO(), &keptnService)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update KeptnService "+keptnService.Name)
		return err
	} else {
		r.ReqLogger.Info("Updated Service " + keptnService.Name)
	}
	return nil
}

func (r *KeptnProjectReconciler) getCommitHash(credentials GitCredentials, branch string) (string, error) {

	authentication := &http.BasicAuth{
		Username: credentials.User,
		Password: credentials.Token,
	}

	cloneOptions := git.CloneOptions{
		URL:  credentials.RemoteURI,
		Auth: authentication,
	}

	if branch != "" {
		cloneOptions = git.CloneOptions{
			URL:           credentials.RemoteURI,
			Auth:          authentication,
			ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		}
	}

	repo, err := git.Clone(memory.NewStorage(), nil, &cloneOptions)
	if err != nil {
		r.ReqLogger.Error(err, "Could not clone repository "+credentials.RemoteURI)
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		r.ReqLogger.Error(err, "Could get head for "+credentials.RemoteURI)
		return "", err
	}
	return head.Hash().String(), nil
}

func (r *KeptnProjectReconciler) getServiceHash(credentials GitCredentials, branch string, service KeptnService) (string, error) {

	authentication := &http.BasicAuth{
		Username: credentials.User,
		Password: credentials.Token,
	}

	cloneOptions := git.CloneOptions{
		URL:           credentials.RemoteURI,
		Auth:          authentication,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
	}

	dir, _ := ioutil.TempDir("", "temp_dir")

	_, err := git.PlainClone(dir, false, &cloneOptions)
	if err != nil {
		r.ReqLogger.Error(err, "Could not checkout "+credentials.RemoteURI)
		return "", err
	}

	hash, err := hashdir.Create(filepath.Join(dir, service.Name, "helm"), "md5")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	return hash, nil
}

func (r *KeptnProjectReconciler) getKeptnServices(project string) keptnv1.KeptnServiceList {
	var keptnServiceList keptnv1.KeptnServiceList

	listOpts := []client.ListOption{
		client.MatchingLabels{"project": project},
	}

	err := r.Client.List(context.TODO(), &keptnServiceList, listOpts...)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get keptn services")
		return keptnServiceList
	}
	return keptnServiceList
}
