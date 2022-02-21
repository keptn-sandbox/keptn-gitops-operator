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
	"github.com/go-logr/logr"
	gitopsv1 "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/api/v1"
	"github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/controllers/common"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"github.com/spf13/afero"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// KeptnGitRepositoryReconciler reconciles a KeptnGitRepository object
type KeptnGitRepositoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	// Recorder contains the Recorder of this controller
	Recorder         record.EventRecorder
	GitClientFactory common.GitClientFactory
}

type KeptnManifests struct {
	projects           []keptnv1.KeptnProject
	services           []keptnv1.KeptnService
	stages             []keptnv1.KeptnStage
	sequences          []keptnv1.KeptnSequence
	execution          []keptnv1.KeptnSequenceExecution
	scheduledexec      []keptnv1.KeptnScheduledExec
	servicedeployments []keptnv1.KeptnServiceDeployment
	instances          []keptnv1.KeptnInstance
}

//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories/finalizers,verbs=update
//+kubebuilder:rbac:groups=keptn.sh,resources=keptngitrepositories/finalizers,verbs=update
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequences/,verbs=get;list
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects/,verbs=get;list
//+kubebuilder:rbac:groups=keptn.sh,resources=keptninstances/,verbs=get;list

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

	keptnGitRepository := &gitopsv1.KeptnGitRepository{}
	err := r.Get(ctx, req.NamespacedName, keptnGitRepository)
	if errors.IsNotFound(err) {
		r.Log.Info("KeptnGitRepository resource not found. Ignoring since object must be deleted")
		return ctrl.Result{}, nil
	}

	r.Log.Info("Syncing", "url", keptnGitRepository.Spec.Repository)
	r.Log.Info("Syncing", "status", keptnGitRepository.Status)

	if keptnGitRepository.Spec.BaseDir == "" {
		keptnGitRepository.Spec.BaseDir = ".keptn"
	}

	if keptnGitRepository.Spec.Branch == "" {
		keptnGitRepository.Spec.Branch = "main"
	}

	fs := afero.NewOsFs()
	codeRepoDir, _ := ioutil.TempDir("", "code_tmp_dir")

	codeRepoConfig, err := common.GetGitCredentials(keptnGitRepository.Spec.Repository, keptnGitRepository.Spec.Username, keptnGitRepository.Spec.Token, keptnGitRepository.Spec.Branch)
	if err != nil {
		r.Log.Error(err, "Could not decode code repo credentials", "URI", keptnGitRepository.Spec.Repository)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	sourceGitClient, err := r.GitClientFactory.GetClient(*codeRepoConfig, codeRepoDir)
	if err != nil {
		r.Log.Error(err, "Could not initialize source git client", "URI", keptnGitRepository.Spec.Repository)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}
	codeRepoHash, err := sourceGitClient.GetLastCommitHash()
	if err != nil {
		r.Log.Error(err, "Could not determine latest commit hash", "URI", keptnGitRepository.Spec.Repository)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if codeRepoHash == keptnGitRepository.Status.LastCommit {
		r.Log.Info("Repository has not changed", "Repository", codeRepoConfig.RemoteURI, "Hash", codeRepoHash)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// TODO abstract file system read operations with an interface
	manifests, err := parseKeptnManifests(codeRepoDir, keptnGitRepository.Spec.BaseDir)
	if err != nil {
		r.Log.Info("Could not parse manifests", "Repository", codeRepoConfig.RemoteURI, "Hash", codeRepoHash)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	for _, instance := range manifests.instances {
		created, err := r.checkCreateInstance(ctx, *keptnGitRepository, instance)
		if err != nil {
			r.Log.Error(err, "Failed to check or create instance")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	for _, sequence := range manifests.sequences {
		err, created := r.checkCreateSequence(ctx, *keptnGitRepository, sequence)
		if err != nil {
			r.Log.Error(err, "Failed to check or create sequence")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	for _, stage := range manifests.stages {
		err, created := r.checkCreateStage(ctx, *keptnGitRepository, stage)
		if err != nil {
			r.Log.Error(err, "Failed to check or create stage")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	for _, project := range manifests.projects {
		err, created := r.checkCreateProject(ctx, *keptnGitRepository, project)
		if err != nil {
			r.Log.Error(err, "Failed to check or create project")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	err = r.deliverArtifacts(ctx, req, fs, keptnGitRepository, codeRepoDir)
	if err != nil {
		r.Log.Error(err, "could not deliver artifacts")
	}

	for _, service := range manifests.services {
		err, created := r.checkCreateService(ctx, *keptnGitRepository, service)
		if err != nil {
			r.Log.Error(err, "Failed to check or create service")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	for _, sequenceexec := range manifests.execution {
		err, created := r.checkCreateSequenceExecution(ctx, *keptnGitRepository, sequenceexec)
		if err != nil {
			r.Log.Error(err, "Failed to check or create sequence execution")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	for _, servicedeployment := range manifests.servicedeployments {
		err, created := r.checkCreateServiceDeployment(ctx, *keptnGitRepository, servicedeployment)
		if err != nil {
			r.Log.Error(err, "Failed to check or create service deployment")
			return ctrl.Result{}, err
		} else if created {
			return ctrl.Result{Requeue: true}, nil
		}
	}

	r.Log.Info("Finished Reconciling")
	r.updateStatusResult(ctx, keptnGitRepository, gitopsv1.KeptnGitRepositoryPhaseSuccessful, codeRepoHash)
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *KeptnGitRepositoryReconciler) updateStatusResult(ctx context.Context, keptnGitRepository *gitopsv1.KeptnGitRepository, result string, hash string) {
	keptnGitRepository.Status.Result = result
	keptnGitRepository.Status.LastCommit = hash
	err := r.Client.Status().Update(ctx, keptnGitRepository)
	if err != nil {
		r.Log.Error(err, "Could not update status", "keptnGitRepository", keptnGitRepository.Name)
	} else {
		r.Log.Info("Updated status", "status", keptnGitRepository.Status)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnGitRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1.KeptnGitRepository{}).
		Complete(r)
}
