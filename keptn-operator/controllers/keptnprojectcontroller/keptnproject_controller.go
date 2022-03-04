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

package keptnprojectcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	nethttp "net/http"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KeptnProjectReconciler reconciles a KeptnProject object
type KeptnProjectReconciler struct {
	client.Client

	// Scheme contains the scheme of this controller
	Scheme *runtime.Scheme
	// Recorder contains the Recorder of this controller
	Recorder record.EventRecorder
	// ReqLogger contains the Logger of this controller
	ReqLogger logr.Logger
	// KeptnInstance contains the Information about the KeptnInstance of this controller
	KeptnInstance apiv1.KeptnInstance
	// KeptnToken contains the API token used in this controller
	KeptnToken string
}

const reconcileErrorInterval = 10 * time.Second
const reconcileSuccessInterval = 120 * time.Second

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnProject object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling Project")

	var err error
	r.KeptnInstance, r.KeptnToken, err = utils.GetKeptnInstance(ctx, r.Client, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get Keptn Instance")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, nil
	}

	keptnproject := &apiv1.KeptnProject{}

	if err := r.Client.Get(ctx, req.NamespacedName, keptnproject); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnProject resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnProject")
		return ctrl.Result{}, err
	}

	myFinalizerName := "keptnprojects.keptn.sh/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if keptnproject.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !utils.ContainsString(keptnproject.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(keptnproject, myFinalizerName)
			if err := r.Update(ctx, keptnproject); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if utils.ContainsString(keptnproject.GetFinalizers(), myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteKeptnProject(keptnproject); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(keptnproject, myFinalizerName)
			if err := r.Update(ctx, keptnproject); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	projectExists, err := utils.CheckKeptnProjectExists(ctx, req, r.Client, keptnproject.Name)
	if !projectExists {
		if keptnproject.Status.ProjectExists {
			r.Recorder.Event(keptnproject, "Warning", "KeptnProjectNotFound", fmt.Sprintf("Keptn project %s does not exist in Keptn", keptnproject.Name))
			keptnproject.Status.ProjectExists = false
			err := r.Client.Status().Update(ctx, keptnproject)
			if err != nil {
				r.ReqLogger.Error(err, "Could not update status of project "+keptnproject.Name)
				return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		err := r.createProject(keptnproject)
		if err != nil {
			r.ReqLogger.Error(err, "Could not create project")
			return ctrl.Result{RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{RequeueAfter: reconcileErrorInterval}, nil
	} else if !keptnproject.Status.ProjectExists {
		keptnproject.Status.ProjectExists = true
		err := r.Client.Status().Update(ctx, keptnproject)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of project "+keptnproject.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	shipyard, err := utils.CreateShipyard(ctx, r.Client, keptnproject.Name)
	if err != nil {
		r.ReqLogger.Error(err, "Could not create shipyard")
		return ctrl.Result{RequeueAfter: reconcileErrorInterval}, err
	}

	shipyardPresent, shipyardHash := utils.CheckKeptnShipyard(ctx, req, r.Client, keptnproject.Name)
	if !shipyardPresent {
		shipyard.Namespace = req.Namespace
		shipyard.Status.LastAppliedHash = utils.GetHashStructure(shipyard.Spec)

		err := controllerutil.SetControllerReference(keptnproject, &shipyard, r.Scheme)
		if err != nil {
			r.ReqLogger.Error(err, "Could not set controller reference")
		}

		err = r.Client.Create(ctx, &shipyard)
		if err != nil {
			r.ReqLogger.Error(err, "Could not create shipyard")
			return ctrl.Result{RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{RequeueAfter: reconcileErrorInterval, Requeue: true}, nil
	}

	err = utils.UpdateShipyard(ctx, r.Client, shipyard, shipyardHash, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update shipyard")
		return ctrl.Result{RequeueAfter: reconcileErrorInterval}, nil
	}

	r.ReqLogger.Info("Finished Reconciling KeptnProject")
	return ctrl.Result{RequeueAfter: reconcileSuccessInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnProject{}).
		Owns(&apiv1.KeptnShipyard{}).
		Complete(r)
}

// Helper functions to check and remove string from a slice of strings.

func (r *KeptnProjectReconciler) deleteKeptnProject(keptnproject *apiv1.KeptnProject) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	request, err := nethttp.NewRequest("DELETE", r.KeptnInstance.Spec.APIUrl+"/controlPlane/v1/project/"+keptnproject.Name, bytes.NewBuffer(nil))
	if err != nil {
		r.ReqLogger.Error(err, "Could not delete Project "+keptnproject.Name)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set(r.KeptnInstance.Status.AuthHeader, r.KeptnToken)

	r.ReqLogger.Info("Deleting Keptn Project " + keptnproject.Name)
	_, err = httpclient.Do(request)
	if err != nil {
		return err
	}
	return err
}

func (r *KeptnProjectReconciler) createProject(project *apiv1.KeptnProject) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	secret, err := utils.DecryptSecret(project.Spec.Password)
	if err != nil {
		r.ReqLogger.Error(err, "could not decrypt secret")
		return err
	}

	data, _ := json.Marshal(map[string]string{
		"gitRemoteURL": project.Spec.Repository,
		"gitToken":     secret,
		"gitUser":      project.Spec.Username,
		"shipyard":     "YXBpVmVyc2lvbjogInNwZWMua2VwdG4uc2gvMC4yLjAiCmtpbmQ6ICJTaGlweWFyZCIKbWV0YWRhdGE6CiAgbmFtZTogInBvZHRhdG8taGVhZCIKc3BlYzoKICBzdGFnZXM6CiAgICAtIG5hbWU6ICJkZXYiCiAgICAgIHNlcXVlbmNlczoKICAgICAgICAtIG5hbWU6ICJkdW1teSIKICAgICAgICAgIHRhc2tzOgogICAgICAgICAgICAtIG5hbWU6ICJkdW1teSIKICAgIC0gbmFtZTogImhhcmRlbmluZyIKICAgICAgc2VxdWVuY2VzOgogICAgICAgIC0gbmFtZTogImR1bW15IgogICAgICAgICAgdGFza3M6CiAgICAgICAgICAgIC0gbmFtZTogImR1bW15IgogICAgLSBuYW1lOiAicHJvZHVjdGlvbiIKICAgICAgc2VxdWVuY2VzOgogICAgICAgIC0gbmFtZTogImR1bW15IgogICAgICAgICAgdGFza3M6CiAgICAgICAgICAgIC0gbmFtZTogImR1bW15IgoK",
		"name":         project.Name,
	})

	request, err := nethttp.NewRequest("POST", r.KeptnInstance.Spec.APIUrl+"/controlPlane/v1/project", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not create project "+project.Name)
		return err
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set(r.KeptnInstance.Status.AuthHeader, r.KeptnToken)

	r.ReqLogger.Info("Creating Keptn Project " + project.Name)
	response, err := httpclient.Do(request)
	if err != nil {
		return err
	}

	err = utils.CheckResponseCode(response, nethttp.StatusOK)
	if err != nil {
		return fmt.Errorf("could not create project %v: %v", project.Name, err)
	}
	return err
}
