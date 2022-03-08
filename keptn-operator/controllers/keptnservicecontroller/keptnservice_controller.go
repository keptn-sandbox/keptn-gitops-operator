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
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	nethttp "net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KeptnServiceReconciler reconciles a KeptnService object
type KeptnServiceReconciler struct {
	client.Client

	// Scheme contains the scheme of this controller
	Scheme *runtime.Scheme
	// Recorder contains the Recorder of this controller
	Recorder record.EventRecorder
	// ReqLogger contains the Logger of this controller
	ReqLogger logr.Logger
	// KeptnInstance contains the Information about the KeptnInstance of this controller
	KeptnInstance apiv1.KeptnInstance
	// KeptnAPIToken contains the API token used in this controller
	KeptnAPIToken string
}

const reconcileErrorInterval = 10 * time.Second
const reconcileSuccessInterval = 120 * time.Second

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

	var err error
	r.KeptnInstance, r.KeptnAPIToken, err = utils.GetKeptnInstance(ctx, r.Client, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get Keptn Instance")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, nil
	}

	keptnservice := &apiv1.KeptnService{}

	if err := r.Client.Get(ctx, req.NamespacedName, keptnservice); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnService resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnService")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, nil
	}

	// name of our custom finalizer
	myFinalizerName := "keptnservices.keptn.sh/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if keptnservice.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !utils.ContainsString(keptnservice.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(keptnservice, myFinalizerName)
			if err := r.Update(ctx, keptnservice); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if utils.ContainsString(keptnservice.GetFinalizers(), myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteKeptnService(keptnservice); err != nil {
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
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if keptnservice.Status.ProjectExists == false {
		keptnservice.Status.ProjectExists = true
		err := r.Client.Status().Update(ctx, keptnservice)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of project "+keptnservice.Spec.Project)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	exists, err := r.checkIfServiceExists(keptnservice.Spec.Project, keptnservice.Spec.Service)
	if !exists {
		err := r.createService(keptnservice.Spec.Service, keptnservice.Spec.Project)
		if err != nil {
			r.ReqLogger.Error(err, "Could not create service "+keptnservice.Spec.Service)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
	}

	r.ReqLogger.Info("Finished Reconciling KeptnService")
	return ctrl.Result{Requeue: true, RequeueAfter: reconcileSuccessInterval}, err
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

	if !projectRes.Status.ProjectExists {
		return false
	}
	return true
}

// Helper functions to check and remove string from a slice of strings.

func (r *KeptnServiceReconciler) deleteKeptnService(keptnservice *apiv1.KeptnService) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	request, err := nethttp.NewRequest("DELETE", r.KeptnInstance.Spec.APIUrl+"/controlPlane/v1/project/"+keptnservice.Spec.Project+"/service/"+keptnservice.Spec.Service, bytes.NewBuffer(nil))
	if err != nil {
		r.ReqLogger.Error(err, "Could not delete service "+keptnservice.Name)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set(r.KeptnInstance.Status.AuthHeader, r.KeptnAPIToken)

	r.ReqLogger.Info("Deleting Keptn Service " + keptnservice.Name)
	_, err = httpclient.Do(request)
	if err != nil {
		return err
	}
	return err
}

func (r *KeptnServiceReconciler) createService(service string, project string) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	data, _ := json.Marshal(map[string]string{
		"serviceName": service,
	})

	request, err := nethttp.NewRequest("POST", r.KeptnInstance.Spec.APIUrl+"/controlPlane/v1/project/"+project+"/service", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not create service "+service)
		return err
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set(r.KeptnInstance.Status.AuthHeader, r.KeptnAPIToken)

	r.ReqLogger.Info("Creating Keptn Service " + service)
	response, err := httpclient.Do(request)
	if err != nil {
		return err
	}
	err = utils.CheckResponseCode(response, nethttp.StatusOK)
	if err != nil {
		return fmt.Errorf("could not create service %v: %v", service, err)
	}

	return err
}

func (r *KeptnServiceReconciler) checkIfServiceExists(project string, service string) (bool, error) {

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)
	servicesHandler := apiutils.NewAuthenticatedServiceHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)

	projects, err := projectsHandler.GetAllProjects()
	if err != nil {
		return false, err
	}

	filteredProjects := utils.FilterProjects(projects, project)
	if len(filteredProjects) == 0 {
		if project != "" {
			return false, fmt.Errorf("no project %s found: %w", project, err)
		}
		return false, fmt.Errorf("no projects found")
	}

	for _, proj := range filteredProjects {
		for _, stage := range proj.Stages {
			services, err := servicesHandler.GetAllServices(proj.ProjectName, stage.StageName)
			if err != nil {
				return false, err
			}
			filteredServices := utils.FilterServices(services, service)
			if len(filteredServices) == 0 {
				return false, fmt.Errorf("no services %s found in project %s", service, project)
			}
			return true, nil
		}
	}
	return false, err
}
