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

package keptnsequenceexecution_controller

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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	nethttp "net/http"
	"os"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KeptnSequenceExecutionReconciler reconciles a KeptnSequenceExecution object
type KeptnSequenceExecutionReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
	ReqLogger      logr.Logger
	keptnApi       string
	keptnApiScheme string
}

type KeptnTriggerEvent struct {
	ContentType string         `json:"contenttype,omitempty"`
	Data        KeptnEventData `json:"data,omitempty"`
	Source      string         `json:"source,omitempty"`
	SpecVersion string         `json:"specversion,omitempty"`
	Type        string         `json:"type,omitempty"`
}

type KeptnEventData struct {
	Project             string                  `json:"project,omitempty"`
	Service             string                  `json:"service,omitempty"`
	Stage               string                  `json:"stage,omitempty"`
	Image               string                  `json:"image,omitempty"`
	Labels              map[string]string       `json:"labels,omitempty"`
	ConfigurationChange ConfigurationChangeData `json:"configurationChange,omitempty"`
}

type ConfigurationChangeData struct {
	Values map[string]string `json:"values,omitempty"`
}

type CreateEventResponse struct {
	KeptnContext string `json:"keptnContext"`
}

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

	var ok bool
	r.keptnApi, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		r.ReqLogger.Info("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.keptnApi = "http://api-gateway-nginx/api"
	}

	if r.keptnApiScheme == "" {
		r.keptnApiScheme = "http"
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

	if !r.checkIfServiceExists(ctx, req, kse.Spec.Project, kse.Spec.Service) {
		r.Recorder.Event(kse, "Warning", "KeptnServiceNotFound", fmt.Sprintf("Keptn service %s in project %s does not exist", kse.Spec.Service, kse.Spec.Project))
		kse.Status.ServiceExists = false
		kse.Status.UpdatePending = true
		fmt.Println(kse.Status.ServiceExists)
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
		err, kcontext := r.triggerTask(ctx, kse, req.Namespace)
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
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
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

func (r *KeptnSequenceExecutionReconciler) checkIfServiceExists(ctx context.Context, req ctrl.Request, project string, service string) bool {

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.keptnApi, utils.GetKeptnToken(r.Client, r.ReqLogger, ctx, req.Namespace), "x-token", nil, r.keptnApiScheme)
	servicesHandler := apiutils.NewAuthenticatedServiceHandler(r.keptnApi, utils.GetKeptnToken(r.Client, r.ReqLogger, ctx, req.Namespace), "x-token", nil, r.keptnApiScheme)

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
		} else {
			fmt.Println("No projects found")
			fmt.Println(err)
			return false
		}
	}

	for _, proj := range filteredProjects {
		fmt.Println(proj.ProjectName)
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

func (r *KeptnSequenceExecutionReconciler) triggerTask(ctx context.Context, exec *apiv1.KeptnSequenceExecution, namespace string) (error, string) {

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

	keptnToken := utils.GetKeptnToken(r.Client, r.ReqLogger, ctx, namespace)

	r.ReqLogger.Info("Triggering Event " + exec.Spec.Event + " for service " + exec.Spec.Service)
	request, err := nethttp.NewRequest("POST", r.keptnApi+"/v1/event", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not trigger event "+exec.Spec.Event+" for service "+exec.Spec.Service)
		return err, ""
	}

	request.Header.Set("content-type", "application/cloudevents+json")
	request.Header.Set("x-token", keptnToken)

	response, err := httpclient.Do(request)
	if err != nil {
		return err, ""
	}

	respBody, err := io.ReadAll(response.Body)
	fmt.Println(string(respBody))

	kcontext := &CreateEventResponse{}

	err = json.Unmarshal(respBody, kcontext)
	if err != nil {
		return err, ""
	}
	return err, kcontext.KeptnContext
}
