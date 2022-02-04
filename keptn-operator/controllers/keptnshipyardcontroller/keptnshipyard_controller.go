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

package keptnshipyardcontroller

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KeptnShipyardReconciler reconciles a KeptnShipyard object
type KeptnShipyardReconciler struct {
	client.Client

	// Scheme contains the scheme of this controller
	Scheme *runtime.Scheme
	// Recorder contains the Recorder of this controller
	Recorder record.EventRecorder
	// ReqLogger contains the Logger of this controller
	ReqLogger logr.Logger
	// KeptnAPI contains the URL of the Keptn Control Plane API
	KeptnAPI string
	// KeptnAPIScheme contains the Scheme (http/https) of the Keptn Control Plane API
	KeptnAPIScheme string
}

const reconcileErrorInterval = 10 * time.Second
const reconcileSuccessInterval = 120 * time.Second

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnshipyards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnshipyards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnshipyards/finalizers,verbs=update
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequences/,verbs=get;list
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects/,verbs=get;list
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnShipyard object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnShipyardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnShipyard")

	var ok bool
	r.KeptnAPI, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		r.ReqLogger.Info("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.KeptnAPI = "http://api-gateway-nginx/api"
	}

	if r.KeptnAPIScheme == "" {
		r.KeptnAPIScheme = "http"
	}

	// your logic here
	shipyardInstance := &apiv1.KeptnShipyard{}
	err := r.Get(ctx, req.NamespacedName, shipyardInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.ReqLogger.Error(err, "Could not fetch shipyard object")
		return reconcile.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}

	shipyardSpecVersion := &v1.ConfigMap{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: "shipyard-" + shipyardInstance.Spec.Project, Namespace: req.Namespace}, shipyardSpecVersion)
	if err != nil {
		if errors.IsNotFound(err) {
			shipyardSpecVersion.Name = "shipyard-" + shipyardInstance.Spec.Project
			shipyardSpecVersion.Namespace = req.Namespace
			shipyardSpecVersion.Data = map[string]string{
				"Hash": "none",
			}
			err := controllerutil.SetControllerReference(shipyardInstance, shipyardSpecVersion, r.Scheme)
			if err != nil {
				r.ReqLogger.Error(err, "could not set controller reference")
				return reconcile.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
			}
			err = r.Client.Create(ctx, shipyardSpecVersion)
			if err != nil {
				r.ReqLogger.Error(err, "Could not create version configmap")
				return reconcile.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	specHash := utils.GetHashStructure(shipyardInstance.Spec)
	if specHash == shipyardSpecVersion.Data["Hash"] {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	projectExists, err := utils.CheckKeptnProjectExists(ctx, req, r.Client, r.KeptnAPI, r.KeptnAPIScheme, shipyardInstance.Spec.Project)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}
	if !projectExists {
		r.Recorder.Event(shipyardInstance, "Warning", "KeptnProjectNotFound", fmt.Sprintf("Keptn project %s does not exist", shipyardInstance.Spec.Project))
		shipyardInstance.Status.ProjectExists = false
		err := r.Client.Status().Update(ctx, shipyardInstance)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of shipyard "+shipyardInstance.Spec.Project)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if shipyardInstance.Status.ProjectExists == false {
		shipyardInstance.Status.ProjectExists = true
		err := r.Client.Status().Update(ctx, shipyardInstance)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of shipyard "+shipyardInstance.Spec.Project)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	keptnShipyard := shipyardInstance.Spec.Shipyard

	shipyardString, err := yaml.Marshal(keptnShipyard)
	if err != nil {
		r.ReqLogger.Error(err, "Could not marshal shipyard")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}

	err = r.updateShipyard(ctx, req.Namespace, shipyardInstance.Spec.Project, shipyardString)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update shipyard")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}

	shipyardSpecVersion.Data["Hash"] = specHash
	err = r.Client.Update(ctx, shipyardSpecVersion)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update status", "KeptnShipyard", shipyardInstance.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	} else {
		r.ReqLogger.Info("Updated status", "status", shipyardInstance.Status)
	}

	r.ReqLogger.Info("Finished Reconciling KeptnShipyard")
	return ctrl.Result{RequeueAfter: reconcileSuccessInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnShipyardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnShipyard{}).
		Complete(r)
}

func (r *KeptnShipyardReconciler) updateShipyard(ctx context.Context, namespace string, project string, shipyard []byte) error {
	upstreamDir, _ := ioutil.TempDir("", "upstream_tmp_dir")

	upstreamRepo, err := utils.GetUpstreamCredentials(ctx, r.Client, project, namespace)
	if err != nil {
		return err
	}

	gitrepo, _, err := utils.CheckOutGitRepo(upstreamRepo, upstreamDir)
	if err != nil {
		return err
	}

	authentication := &githttp.BasicAuth{
		Username: upstreamRepo.User,
		Password: upstreamRepo.Token,
	}

	commitOptions := git.CommitOptions{
		Author: &object.Signature{
			Name:  "Keptn Upstream Pusher",
			Email: "keptn@keptn.sh",
			When:  time.Now(),
		},
	}

	err = ioutil.WriteFile(filepath.Join(upstreamDir, "shipyard.yaml"), shipyard, 0444)
	if err != nil {
		return fmt.Errorf("could not write shipyard: %w", err)
	}

	w, err := gitrepo.Worktree()
	if err != nil {
		return fmt.Errorf("could not set worktree: %w", err)
	}

	// go-git can't stage deleted files https://github.com/src-d/go-git/issues/1268
	err = utils.AddGit(w)
	if err != nil {
		return fmt.Errorf("could not add files: %w", err)
	}

	_, err = w.Commit("Push new version", &commitOptions)
	if err != nil {
		return fmt.Errorf("could not commit: %w", err)
	}

	err = gitrepo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       authentication,
		RefSpecs: []config.RefSpec{
			"refs/heads/*:refs/heads/*",
		},
	})
	if err != nil {
		return fmt.Errorf("could not push commit: %w", err)
	}
	return nil
}
