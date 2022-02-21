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

package keptnservicedeploymentcontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"github.com/keptn/go-utils/pkg/api/models"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	nethttp "net/http"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KeptnServiceDeploymentReconciler reconciles a KeptnServiceDeployment object
type KeptnServiceDeploymentReconciler struct {
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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicedeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicedeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicedeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnServiceDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnServiceDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnSequenceExecution")

	var err error
	r.KeptnInstance, r.KeptnAPIToken, err = utils.GetKeptnInstance(ctx, r.Client, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get Keptn Instance")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, nil
	}

	ksd := &apiv1.KeptnServiceDeployment{}

	if err := r.Client.Get(ctx, req.NamespacedName, ksd); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnServiceDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnServiceDeployment")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}

	if !r.checkKeptnProject(ctx, req, ksd.Spec.Project) {
		r.Recorder.Event(ksd, "Warning", "KeptnProjectNotFound", fmt.Sprintf("Keptn project %s does not exist", ksd.Spec.Project))
		ksd.Status.Prerequisites.ProjectExists = false
		ksd.Status.UpdatePending = true
		err := r.Client.Status().Update(ctx, ksd)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of KeptnServiceDeployment "+ksd.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if ksd.Status.Prerequisites.ProjectExists == false {
		ksd.Status.Prerequisites.ProjectExists = true
		err := r.Client.Status().Update(ctx, ksd)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of KeptnServiceDeployment "+ksd.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}

	service, _, serviceExists := r.checkIfServiceExists(ctx, req, ksd.Spec.Project, ksd.Spec.Service)
	if !serviceExists {
		r.Recorder.Event(ksd, "Warning", "KeptnServiceNotFound", fmt.Sprintf("Keptn service %s in project %s does not exist", ksd.Spec.Service, ksd.Spec.Project))
		ksd.Status.Prerequisites.ServiceExists = false
		ksd.Status.UpdatePending = true
		err := r.Client.Status().Update(ctx, ksd)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of ksd "+ksd.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if ksd.Status.Prerequisites.ServiceExists == false {
		ksd.Status.Prerequisites.ServiceExists = true
		err := r.Client.Status().Update(ctx, ksd)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of ksd "+ksd.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileSuccessInterval}, nil
	}

	keptncontext, newcontext, err := getKeptnContext(r.Client, ctx, req.Namespace, ksd.Spec.Project, ksd.Spec.Service, ksd.Spec.Version)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get KeptnContext for Service Deployment "+ksd.Name)
	}

	if newcontext {
		newContext := apiv1.KeptnDeploymentContext{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ksd.Spec.Project + "-" + ksd.Spec.Service + "-" + ksd.Spec.Version,
				Namespace: req.Namespace,
			},
			Spec:   apiv1.KeptnDeploymentContextSpec{},
			Status: apiv1.KeptnDeploymentContextStatus{},
		}
		err := controllerutil.SetControllerReference(&service, &newContext, r.Scheme)
		if err != nil {
			r.ReqLogger.Error(err, "could not set controller reference:")
		}
		err = r.Client.Create(ctx, &newContext)
		if err != nil {
			r.ReqLogger.Error(err, "Could not create deployment context")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if keptncontext.Status.LastAppliedHash == nil {
		keptncontext.Status.LastAppliedHash = make(map[string]string)
	}

	if keptncontext.Status.LastAppliedHash[ksd.Spec.Stage] != utils.GetHashStructure(ksd.Spec) || ksd.Status.UpdatePending {
		kcontext, err := r.triggerTask(ksd, service.Spec.DeploymentEvent, keptncontext.Status.KeptnContext)
		if err != nil {
			r.ReqLogger.Error(err, "Could not trigger task")
			return ctrl.Result{Requeue: true}, err
		}
		keptncontext.Status.KeptnContext = kcontext
		keptncontext.Status.LastAppliedHash[ksd.Spec.Stage] = utils.GetHashStructure(ksd.Spec)
		err = r.Client.Status().Update(ctx, keptncontext)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of ksd "+ksd.Name)
		}

		ksd.Status.UpdatePending = false
		ksd.Status.KeptnContext = kcontext
		ksd.Status.LastAppliedHash = utils.GetHashStructure(ksd.Spec)
		err = r.Client.Status().Update(ctx, ksd)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of ksd "+ksd.Name)
		}
	}
	r.ReqLogger.Info("Finished Reconciling KeptnSequenceExecution")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnServiceDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnServiceDeployment{}).
		Complete(r)
}

func (r *KeptnServiceDeploymentReconciler) checkKeptnProject(ctx context.Context, req ctrl.Request, project string) bool {
	projectRes := &apiv1.KeptnProject{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: project, Namespace: req.Namespace}, projectRes)
	if err != nil {
		return false
	}

	return true
}

func (r *KeptnServiceDeploymentReconciler) checkIfServiceExists(ctx context.Context, req ctrl.Request, project string, service string) (kservice apiv1.KeptnService, stages []*models.Stage, exists bool) {
	serviceRes, err := r.servicesList(ctx, req, project, service)
	if err != nil {
		return serviceRes, nil, false
	}

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)
	servicesHandler := apiutils.NewAuthenticatedServiceHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)

	projects, err := projectsHandler.GetAllProjects()
	if err != nil {
		return serviceRes, nil, false
	}

	filteredProjects := utils.FilterProjects(projects, project)
	if len(filteredProjects) == 0 {
		if project != "" {
			r.ReqLogger.Info(fmt.Sprintf("No project %s found", project))
			return serviceRes, nil, false
		}
		return serviceRes, nil, false
	}

	for _, proj := range filteredProjects {
		for _, stage := range proj.Stages {
			services, err := servicesHandler.GetAllServices(proj.ProjectName, stage.StageName)
			if err != nil {
				return serviceRes, nil, false
			}
			filteredServices := utils.FilterServices(services, service)
			if len(filteredServices) == 0 {
				r.ReqLogger.Info(fmt.Sprintf("No services %s found in project %s", service, project))
				return serviceRes, nil, false
			}
			return serviceRes, proj.Stages, true
		}
	}
	return serviceRes, nil, false
}

func getKeptnContext(client client.Client, ctx context.Context, namespace string, project string, service string, version string) (*apiv1.KeptnDeploymentContext, bool, error) {
	found := &apiv1.KeptnDeploymentContext{}
	err := client.Get(ctx, types.NamespacedName{Name: project + "-" + service + "-" + version, Namespace: namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			return found, true, nil
		}
		return found, false, err
	}
	return found, false, nil
}

func (r *KeptnServiceDeploymentReconciler) triggerTask(deployment *apiv1.KeptnServiceDeployment, deploymentEvent string, shkeptncontext string) (string, error) {

	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	event := KeptnTriggerEvent{
		ContentType: "application/json",
		Data: KeptnEventData{
			Service: deployment.Spec.Service,
			Project: deployment.Spec.Project,
			Stage:   deployment.Spec.Stage,
			Labels: map[string]string{
				"version":          deployment.Spec.Version,
				"author":           deployment.Spec.Author,
				"sourceCommitHash": deployment.Spec.SourceCommitHash,
			},
			Image: deployment.Spec.Service + ":" + deployment.Spec.Version,
		},
		Source:      "Keptn GitOps Operator",
		SpecVersion: "1.0",
		Type:        "sh.keptn.event." + deployment.Spec.Stage + "." + deploymentEvent,
	}

	if shkeptncontext != "" {
		event.Context = shkeptncontext
	}

	data, err := json.Marshal(event)
	if err != nil {
		r.ReqLogger.Info("Could not marshal Keptn Trigger Event")
	}

	r.ReqLogger.Info("Triggering Event sh.keptn.event." + deployment.Spec.Stage + "." + deploymentEvent + " for service " + deployment.Spec.Service)
	request, err := nethttp.NewRequest("POST", r.KeptnInstance.Spec.APIUrl+"/v1/event", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not trigger event "+deployment.Spec.Stage+"."+deploymentEvent+" for service "+deployment.Spec.Service)
		return "", err
	}

	request.Header.Set("content-type", "application/cloudevents+json")
	request.Header.Set(r.KeptnInstance.Status.AuthHeader, r.KeptnAPIToken)

	response, err := httpclient.Do(request)
	if err != nil {
		return "", err
	}

	err = utils.CheckResponseCode(response, nethttp.StatusOK)
	if err != nil {
		return "", err
	}

	respBody, err := io.ReadAll(response.Body)
	kcontext := &CreateEventResponse{}

	err = json.Unmarshal(respBody, kcontext)
	if err != nil {
		return "", err
	}
	return kcontext.KeptnContext, err
}

func (r *KeptnServiceDeploymentReconciler) servicesList(ctx context.Context, req ctrl.Request, project string, service string) (apiv1.KeptnService, error) {
	serviceList := &apiv1.KeptnServiceList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}
	err := r.Client.List(ctx, serviceList, opts...)
	if err != nil {
		return apiv1.KeptnService{}, err
	}
	if len(serviceList.Items) == 0 {
		return apiv1.KeptnService{}, fmt.Errorf("no service found")
	}
	for _, svc := range serviceList.Items {
		if svc.Spec.Project == project && svc.Spec.Service == service {
			return svc, nil
		}
	}
	return apiv1.KeptnService{}, fmt.Errorf("no service found")
}
