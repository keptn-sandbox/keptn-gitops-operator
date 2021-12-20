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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/imroc/req"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	nethttp "net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	keptnapi "github.com/keptn/go-utils/pkg/api/models"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	ctrl "sigs.k8s.io/controller-runtime"
)

const defaultKeptnControlPlaneAPIURL = "http://shipyard-controller.keptn:8080"

const shipyardAPIVersion = "spec.keptn.sh/0.2.2"

// KeptnShipyardReconciler reconciles a KeptnShipyard object
type KeptnShipyardReconciler struct {
	utils.KeptnReconcile
}

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnshipyards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnshipyards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnshipyards/finalizers,verbs=update
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequences/,verbs=get;list
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects/,verbs=get;list
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;

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
		return reconcile.Result{Requeue: true}, err
	}

	if !r.checkKeptnProjectExists(ctx, req, shipyardInstance.Spec.Project) {
		r.Recorder.Event(shipyardInstance, "Warning", "KeptnProjectNotFound", fmt.Sprintf("Keptn project %s does not exist", shipyardInstance.Spec.Project))
		shipyardInstance.Status.ProjectExists = false
		err := r.Client.Status().Update(ctx, shipyardInstance)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of shipyard "+shipyardInstance.Spec.Project)
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	} else if shipyardInstance.Status.ProjectExists == false {
		shipyardInstance.Status.ProjectExists = true
		err := r.Client.Status().Update(ctx, shipyardInstance)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update status of shipyard "+shipyardInstance.Spec.Project)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	keptnShipyard := keptnv2.Shipyard{}
	keptnShipyard.Kind = shipyardInstance.Kind
	keptnShipyard.ApiVersion = shipyardAPIVersion
	stages, err := r.transformStageSpecsToKeptnAPI(shipyardInstance, shipyardInstance.Spec.Stages)
	if err != nil {
		r.ReqLogger.Error(err, "Could not transform stages")
		return reconcile.Result{Requeue: true}, err
	}
	keptnShipyard.Spec.Stages = stages

	shipyardString, err := yaml.Marshal(keptnShipyard)
	if err != nil {
		r.ReqLogger.Error(err, "Could not marshal shipyard")
		return reconcile.Result{Requeue: true}, err
	}
	// encodedShipyardString := base64.StdEncoding.EncodeToString(shipyardString)

	newProject := apiv1.CreateProject{
		Name:     &shipyardInstance.Spec.Project,
		Shipyard: shipyardString,
	}

	_, err = r.updateShipyard(ctx, req.Namespace, shipyardInstance.Spec.Project, newProject)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	r.ReqLogger.Info("Finished Reconciling KeptnShipyard")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *KeptnShipyardReconciler) transformStageSpecsToKeptnAPI(instance *apiv1.KeptnShipyard, stages []apiv1.KeptnShipyardStage) ([]keptnv2.Stage, error) {
	result := []keptnv2.Stage{}

	allSequences := &apiv1.KeptnSequenceList{}
	err := r.List(context.TODO(), allSequences)
	if err != nil {
		return nil, fmt.Errorf("could not load available sequnces: %w", err)
	}

	for _, stageRef := range stages {
		stageInstance := &apiv1.KeptnStage{}
		if err := r.Get(context.TODO(), types.NamespacedName{
			Namespace: instance.Namespace,
			Name:      stageRef.StageRef,
		}, stageInstance); err != nil {
			return nil, fmt.Errorf("could not load stage: %w", err)
		}
		newStage := keptnv2.Stage{Name: stageInstance.Name, Sequences: []keptnv2.Sequence{}}

		// get the referenced sequences
		for _, seq := range stageInstance.Spec.Sequence {
			sequenceFound := false
			for _, availableSequence := range allSequences.Items {
				if availableSequence.Name == seq.SequenceRef {
					sequenceFound = true

					keptnv2Sequence := &keptnv2.Sequence{}
					if err := keptnv2.Decode(availableSequence.Spec.Sequence, &keptnv2Sequence); err != nil {
						return nil, fmt.Errorf("could not transform sequence: %w", err)
					}
					newStage.Sequences = append(newStage.Sequences, *keptnv2Sequence)
					break
				}
			}
			if !sequenceFound {
				return nil, fmt.Errorf("could not find sequence %s", seq.SequenceRef)
			}
		}
		result = append(result, newStage)
	}

	return result, nil
}

func (r *KeptnShipyardReconciler) fetchProject(err error, shipyardInstance *apiv1.KeptnShipyard, logger logr.Logger) (*keptnapi.Project, error) {
	get, err := req.Get(fmt.Sprintf("%s/v1/project/%s", getKeptnAPIURL(), shipyardInstance.Spec.Project))
	if err != nil {

		return nil, fmt.Errorf("could not fetch projects from Keptn API: %w", err)
	}

	project := &keptnapi.Project{}
	if err := get.ToJSON(project); err != nil {
		return nil, fmt.Errorf("could not parse API response: %w", err)
	}

	return project, nil
}

func getKeptnAPIURL() interface{} {
	if apiURL := os.Getenv("KEPTN_CONTROL_PLANE_API_URL"); apiURL != "" {
		return apiURL
	}
	return defaultKeptnControlPlaneAPIURL
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnShipyardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnShipyard{}).
		Complete(r)
}

func (r *KeptnShipyardReconciler) checkKeptnProjectExists(ctx context.Context, req ctrl.Request, project string) bool {

	projectsHandler := apiutils.NewAuthenticatedProjectHandler(r.KeptnAPI, utils.GetKeptnToken(r.Client, r.ReqLogger, ctx, req.Namespace), "x-token", nil, r.KeptnAPIScheme)

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
	return true
}

func (r *KeptnShipyardReconciler) updateShipyard(ctx context.Context, namespace string, project string, createProject apiv1.CreateProject) (int, error) {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	data, _ := json.Marshal(createProject)

	keptnToken := utils.GetKeptnToken(r.Client, r.ReqLogger, ctx, namespace)

	request, err := nethttp.NewRequest("PUT", r.KeptnAPI+"/controlPlane/v1/project", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not update shipyard for project "+project)
		return 0, err
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", keptnToken)

	r.ReqLogger.Info("Updating Shipyard for project " + project)
	response, err := httpclient.Do(request)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, err
}
