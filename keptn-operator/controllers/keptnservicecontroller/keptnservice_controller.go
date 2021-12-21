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

package keptnservicecontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	nethttp "net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KeptnServiceReconciler reconciles a KeptnService object
type KeptnServiceReconciler struct {
	utils.KeptnReconcile
}

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnService")

	var ok bool
	r.KeptnAPI, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		r.ReqLogger.Info("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.KeptnAPI = "http://api-gateway-nginx/api"
	}

	if r.KeptnAPIScheme == "" {
		r.KeptnAPIScheme = "http"
	}
	keptnservice := &apiv1.KeptnService{}

	if err := r.Client.Get(ctx, req.NamespacedName, keptnservice); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnService resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnService")
		return ctrl.Result{}, err
	}

	// name of our custom finalizer
	myFinalizerName := "keptnservices.keptn.sh/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if keptnservice.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(keptnservice.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(keptnservice, myFinalizerName)
			if err := r.Update(ctx, keptnservice); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(keptnservice.GetFinalizers(), myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteKeptnService(ctx, req.Namespace, keptnservice); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(keptnservice, myFinalizerName)
			if err := r.Update(ctx, keptnservice); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if !r.checkKeptnProject(ctx, req, keptnservice.Spec.Project) {
		r.Recorder.Event(keptnservice, "Warning", "KeptnProjectNotFound", fmt.Sprintf("Keptn project %s does not exist", keptnservice.Spec.Project))
		keptnservice.Status.ProjectExists = false
		err := r.Client.Status().Update(ctx, keptnservice)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of project "+keptnservice.Spec.Project)
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if keptnservice.Status.ProjectExists == false {
		keptnservice.Status.ProjectExists = true
		err := r.Client.Status().Update(ctx, keptnservice)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of project "+keptnservice.Spec.Project)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !r.checkIfServiceExists(ctx, req, keptnservice.Spec.Project, keptnservice.Name) {
		_, err := r.createService(ctx, keptnservice.Name, req.Namespace, keptnservice.Spec.Project)
		if err != nil {
			fmt.Println("Could not create service")
		}
	}

	r.ReqLogger.Info("Finished Reconciling KeptnService")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnService{}).
		Complete(r)
}

func (r *KeptnServiceReconciler) checkKeptnProject(ctx context.Context, req ctrl.Request, project string) bool {
	projectRes := &apiv1.KeptnProject{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: project, Namespace: req.Namespace}, projectRes)
	if err != nil {
		return false
	}

	return true
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func (r *KeptnServiceReconciler) deleteKeptnService(ctx context.Context, namespace string, keptnservice *apiv1.KeptnService) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	keptnToken := utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, namespace)

	request, err := nethttp.NewRequest("DELETE", r.KeptnAPI+"/controlPlane/v1/project/"+keptnservice.Spec.Project+"/service/"+keptnservice.Name, bytes.NewBuffer(nil))
	if err != nil {
		r.ReqLogger.Error(err, "Could not delete service "+keptnservice.Name)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", keptnToken)

	r.ReqLogger.Info("Deleting Keptn Service " + keptnservice.Name)
	_, err = httpclient.Do(request)
	if err != nil {
		return err
	}
	return err
}

func (r *KeptnServiceReconciler) createService(ctx context.Context, service string, namespace string, project string) (int, error) {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	data, _ := json.Marshal(map[string]string{
		"serviceName": service,
	})

	keptnToken := utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, namespace)

	request, err := nethttp.NewRequest("POST", r.KeptnAPI+"/controlPlane/v1/project/"+project+"/service", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not create service "+service)
		return 0, err
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", keptnToken)

	r.ReqLogger.Info("Creating Keptn Service " + service)
	response, err := httpclient.Do(request)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, err
}

func (r *KeptnServiceReconciler) checkIfServiceExists(ctx context.Context, req ctrl.Request, project string, service string) bool {

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.KeptnAPI, utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, req.Namespace), "x-token", nil, r.KeptnAPIScheme)
	servicesHandler := apiutils.NewAuthenticatedServiceHandler(r.KeptnAPI, utils.GetKeptnToken(ctx, r.Client, r.ReqLogger, req.Namespace), "x-token", nil, r.KeptnAPIScheme)

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

	fmt.Println(filteredProjects)

	for _, proj := range filteredProjects {
		for _, stage := range proj.Stages {
			services, err := servicesHandler.GetAllServices(proj.ProjectName, stage.StageName)
			if err != nil {
				return false
			}
			filteredServices := utils.FilterServices(services, service)
			if len(filteredServices) == 0 {
				fmt.Printf("No services %s found in project %s", service, project)
				return false
			}
			return true
		}
	}
	return false
}
