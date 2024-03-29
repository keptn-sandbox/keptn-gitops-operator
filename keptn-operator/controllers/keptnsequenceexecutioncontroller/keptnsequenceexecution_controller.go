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

package keptnsequenceexecutioncontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	nethttp "net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KeptnSequenceExecutionReconciler reconciles a KeptnSequenceExecution object
type KeptnSequenceExecutionReconciler struct {
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

// KeptnTriggerEvent describes a Keptn Event which should be triggered
type KeptnTriggerEvent struct {
	ContentType string         `json:"contenttype,omitempty"`
	Data        KeptnEventData `json:"data,omitempty"`
	Source      string         `json:"source,omitempty"`
	SpecVersion string         `json:"specversion,omitempty"`
	Type        string         `json:"type,omitempty"`
}

// KeptnEventData describes the Event Data of an KeptnTriggerEvent
type KeptnEventData struct {
	Project             string                  `json:"project,omitempty"`
	Service             string                  `json:"service,omitempty"`
	Stage               string                  `json:"stage,omitempty"`
	Image               string                  `json:"image,omitempty"`
	Labels              map[string]string       `json:"labels,omitempty"`
	ConfigurationChange ConfigurationChangeData `json:"configurationChange,omitempty"`
}

// ConfigurationChangeData describes the Configuration Change block of a KeptnEventData
type ConfigurationChangeData struct {
	Values map[string]string `json:"values,omitempty"`
}

// CreateEventResponse describes the Response of a sent Keptn Event
type CreateEventResponse struct {
	KeptnContext string `json:"keptnContext"`
}

const reconcileErrorInterval = 10 * time.Second
const reconcileSuccessInterval = 120 * time.Second

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequenceexecutions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequenceexecutions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequenceexecutions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnSequenceExecution object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnSequenceExecutionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnSequenceExecution")

	var err error
	r.KeptnInstance, r.KeptnAPIToken, err = utils.GetKeptnInstance(ctx, r.Client, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get Keptn Instance")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, nil
	}

	kse := &apiv1.KeptnSequenceExecution{}

	if err := r.Client.Get(ctx, req.NamespacedName, kse); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnSequenceExecution resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnService")
		return ctrl.Result{}, err
	}

	if !r.checkKeptnProject(ctx, req, kse.Spec.Project) {
		r.Recorder.Event(kse, "Warning", "KeptnProjectNotFound", fmt.Sprintf("Keptn project %s does not exist", kse.Spec.Project))
		kse.Status.ProjectExists = false
		kse.Status.UpdatePending = true
		err := r.Client.Status().Update(ctx, kse)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of KeptnSequenceExecution "+kse.Name)
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if kse.Status.ProjectExists == false {
		kse.Status.ProjectExists = true
		err := r.Client.Status().Update(ctx, kse)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of KeptnSequenceExecution "+kse.Name)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	exists, err := r.checkIfServiceExists(kse.Spec.Project, kse.Spec.Service)
	if !exists {
		r.Recorder.Event(kse, "Warning", "KeptnServiceNotFound", fmt.Sprintf("Keptn service %s in project %s does not exist", kse.Spec.Service, kse.Spec.Project))
		kse.Status.ServiceExists = false
		kse.Status.UpdatePending = true
		err := r.Client.Status().Update(ctx, kse)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of kse "+kse.Name)
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if kse.Status.ServiceExists == false {
		kse.Status.ServiceExists = true
		err := r.Client.Status().Update(ctx, kse)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of kse "+kse.Name)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if kse.Status.KeptnContext == "" || kse.Status.LastAppliedHash != utils.GetHashStructure(kse.Spec) || kse.Status.UpdatePending {
		kcontext, err := r.triggerTask(kse)
		if err != nil {
			r.ReqLogger.Error(err, "Could not trigger task")
			return ctrl.Result{Requeue: true}, err
		}
		kse.Status.UpdatePending = false
		kse.Status.KeptnContext = kcontext
		kse.Status.LastAppliedHash = utils.GetHashStructure(kse.Spec)
		err = r.Client.Status().Update(ctx, kse)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of kse "+kse.Name)
		}
	}

	r.ReqLogger.Info("Finished Reconciling KeptnSequenceExecution")
	return ctrl.Result{RequeueAfter: reconcileSuccessInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnSequenceExecutionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnSequenceExecution{}).
		Complete(r)
}

func (r *KeptnSequenceExecutionReconciler) checkKeptnProject(ctx context.Context, req ctrl.Request, project string) bool {
	projectRes := &apiv1.KeptnProject{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: project, Namespace: req.Namespace}, projectRes)
	if err != nil {
		return false
	}

	return true
}

func (r *KeptnSequenceExecutionReconciler) checkIfServiceExists(project string, service string) (bool, error) {

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)
	servicesHandler := apiutils.NewAuthenticatedServiceHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)

	projects, err := projectsHandler.GetAllProjects()
	if err != nil {
		return false, err
	}

	filteredProjects := utils.FilterProjects(projects, project)
	if len(filteredProjects) == 0 {
		if project != "" {
			return false, fmt.Errorf("no project %s found", project)
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
	return false, nil
}

func (r *KeptnSequenceExecutionReconciler) triggerTask(exec *apiv1.KeptnSequenceExecution) (string, error) {

	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	version := "undefined"

	if exec.Spec.Labels["version"] != "" {
		version = exec.Spec.Labels[version]
	}

	data, err := json.Marshal(KeptnTriggerEvent{
		ContentType: "application/json",
		Data: KeptnEventData{
			Service: exec.Spec.Service,
			Project: exec.Spec.Project,
			Stage:   exec.Spec.Stage,
			Labels:  exec.Spec.Labels,
			Image:   exec.Spec.Service + ":" + version,
		},
		Source:      "Keptn GitOps Operator",
		SpecVersion: "1.0",
		Type:        "sh.keptn.event." + exec.Spec.Event,
	})
	if err != nil {
		r.ReqLogger.Info("Could not marshal Keptn Trigger Event")
	}

	r.ReqLogger.Info("Triggering Event " + exec.Spec.Event + " for service " + exec.Spec.Service)
	request, err := nethttp.NewRequest("POST", r.KeptnInstance.Spec.APIUrl+"/v1/event", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not trigger event "+exec.Spec.Event+" for service "+exec.Spec.Service)
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
