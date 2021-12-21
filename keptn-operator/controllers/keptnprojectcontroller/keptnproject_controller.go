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
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	nethttp "net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KeptnProjectReconciler reconciles a KeptnProject object
type KeptnProjectReconciler struct {
	utils.KeptnReconcile
}

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

	var ok bool
	r.KeptnAPI, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		r.ReqLogger.Info("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.KeptnAPI = "http://api-gateway-nginx/api"
	}

	if r.KeptnAPIScheme == "" {
		r.KeptnAPIScheme = "http"
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
			if err := r.deleteKeptnProject(ctx, req.Namespace, keptnproject); err != nil {
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

	if !r.checkKeptnProjectExists(ctx, req, keptnproject.Name) {
		_, err := r.createProject(ctx, keptnproject, req.Namespace)
		if err != nil {
			fmt.Println("Could not create project")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	shipyard := r.createShipyard(ctx, keptnproject)
	shipyardPresent, shipyardHash := checkKeptnShipyard(ctx, req, r.Client, keptnproject.Name)
	if !shipyardPresent {
		shipyard.Namespace = req.Namespace
		shipyard.Status.LastAppliedHash = utils.GetHashStructure(shipyard.Spec)

		err := r.Client.Create(ctx, &shipyard)
		if err != nil {
			fmt.Println("Could not create Shipyard")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	shipyard.Namespace = req.Namespace
	shipyard.Status.LastAppliedHash = utils.GetHashStructure(shipyard.Spec)

	if utils.GetHashStructure(shipyard.Spec) != shipyardHash {
		currentShipyard := &apiv1.KeptnShipyard{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: keptnproject.Name, Namespace: req.Namespace}, currentShipyard)

		if err != nil {
			r.ReqLogger.Error(err, "Could not get shipyard "+shipyard.Name)
		}

		currentShipyard.Spec = shipyard.Spec
		currentShipyard.Status.LastAppliedHash = utils.GetHashStructure(currentShipyard.Spec)
		err = r.Client.Update(ctx, currentShipyard)

		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of shipyard "+currentShipyard.Name)
		}
	}

	r.ReqLogger.Info("Finished Reconciling KeptnProject")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnProject{}).
		Complete(r)
}

func (r *KeptnProjectReconciler) checkKeptnProjectExists(ctx context.Context, req ctrl.Request, project string) bool {

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.KeptnAPI, utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, req.Namespace), "x-token", nil, r.KeptnAPIScheme)

	projects, err := projectsHandler.GetAllProjects()
	if err != nil {
		fmt.Println(err)
		return false
	}

	filteredProjects := utils.FilterProjects(projects, project)
	if len(filteredProjects) == 0 {
		if project != "" {
			fmt.Printf("No project %s found\n", project)
			fmt.Println(err)
			return false
		}
		fmt.Println("No projects found")
		fmt.Println(err)
		return false
	}
	return true
}

// Helper functions to check and remove string from a slice of strings.

func (r *KeptnProjectReconciler) deleteKeptnProject(ctx context.Context, namespace string, keptnproject *apiv1.KeptnProject) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	keptnToken := utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, namespace)

	request, err := nethttp.NewRequest("DELETE", r.KeptnAPI+"/controlPlane/v1/project/"+keptnproject.Name, bytes.NewBuffer(nil))
	if err != nil {
		r.ReqLogger.Error(err, "Could not delete Project "+keptnproject.Name)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", keptnToken)

	r.ReqLogger.Info("Deleting Keptn Project " + keptnproject.Name)
	_, err = httpclient.Do(request)
	if err != nil {
		return err
	}
	return err
}

func (r *KeptnProjectReconciler) createProject(ctx context.Context, project *apiv1.KeptnProject, namespace string) (int, error) {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	secret, err := decryptSecret(project.Spec.Password)
	if err != nil {
		fmt.Println("could not decrypt secret")
		return 0, err
	}

	data, _ := json.Marshal(map[string]string{
		"gitRemoteURL": project.Spec.Repository,
		"gitToken":     secret,
		"gitUser":      project.Spec.Username,
		"shipyard":     "YXBpVmVyc2lvbjogInNwZWMua2VwdG4uc2gvMC4yLjAiCmtpbmQ6ICJTaGlweWFyZCIKbWV0YWRhdGE6CiAgbmFtZTogInBvZHRhdG8taGVhZCIKc3BlYzoKICBzdGFnZXM6CiAgICAtIG5hbWU6ICJkdW1teSIKICAgICAgc2VxdWVuY2VzOgogICAgICAgIC0gbmFtZTogImR1bW15IgogICAgICAgICAgdGFza3M6CiAgICAgICAgICAgIC0gbmFtZTogImR1bW15Igo=",
		"name":         project.Name,
	})

	keptnToken := utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, namespace)

	request, err := nethttp.NewRequest("POST", r.KeptnAPI+"/controlPlane/v1/project", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not create project "+project.Name)
		return 0, err
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", keptnToken)

	r.ReqLogger.Info("Creating Keptn Project " + project.Name)
	response, err := httpclient.Do(request)
	if err != nil {
		return 0, err
	}

	respBody, _ := ioutil.ReadAll(response.Body)
	fmt.Println(string(respBody))

	return response.StatusCode, err
}

func checkKeptnShipyard(ctx context.Context, req ctrl.Request, client client.Client, project string) (bool, string) {
	shipyardRes := &apiv1.KeptnShipyard{}

	err := client.Get(ctx, types.NamespacedName{Name: project, Namespace: req.Namespace}, shipyardRes)
	if err != nil {
		return false, ""
	}
	return true, shipyardRes.Status.LastAppliedHash
}
