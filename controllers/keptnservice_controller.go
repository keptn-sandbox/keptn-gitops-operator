/*


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
	"bytes"
	"context"
	"encoding/json"
	nethttp "net/http"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keptnv1 "keptn-operator/api/v1"
)

// KeptnServiceReconciler reconciles a KeptnService object
type KeptnServiceReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	keptnApi  string
	ReqLogger logr.Logger
}

// +kubebuilder:rbac:groups=keptn.operator.keptn.sh,resources=keptnservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keptn.operator.keptn.sh,resources=keptnservices/status,verbs=get;update;patch

func (r *KeptnServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	r.ReqLogger = r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnService")

	var ok bool
	r.keptnApi, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		r.ReqLogger.Info("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.keptnApi = "http://api-gateway-nginx/api"
	}

	service := &keptnv1.KeptnService{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, service)
	if errors.IsNotFound(err) {
		r.ReqLogger.Info("KeptnProject resource not found. Ignoring since object must be deleted")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	if service.Status.CreationPending && !r.checkKeptnServiceExists(service, req.Namespace) {
		service.Status.LastSetupStatus, err = r.createService(service.Spec.Service, req.Namespace, service.Spec.Project)
		if err != nil {
			r.ReqLogger.Error(err, "Could not create service "+service.Spec.Service)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		service.Status.CreationPending = false
	}

	if service.Status.DeploymentPending {
		r.ReqLogger.Info("Deployment is pending")
		err = r.triggerDeployment(service.Spec.Service, req.Namespace, service.Spec.Project, service.Spec.StartStage, service.Spec.TriggerCommand)
		if err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Second}, err
		}
		service.Status.DeploymentPending = false
		err = r.Client.Update(context.TODO(), service)
		if err != nil {
			r.ReqLogger.Error(err, "Could not update Service")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		}
		return ctrl.Result{}, nil
	}

	if service.Status.DeletionPending {
		r.ReqLogger.Info("Deletion is pending")
		err = r.deleteService(service.Spec.Service, req.Namespace, service.Spec.Project)
		if err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Second}, err
		}
		service.Status.SafeToDelete = true
	}

	err = r.Client.Update(context.TODO(), service)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update Service")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	r.ReqLogger.Info("Finished Reconciling")

	return ctrl.Result{RequeueAfter: 180 * time.Second}, nil
}

func (r *KeptnServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keptnv1.KeptnService{}).
		Complete(r)
}

func (r *KeptnServiceReconciler) createService(service string, namespace string, project string) (int, error) {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	data, _ := json.Marshal(map[string]string{
		"serviceName": service,
	})

	keptnToken := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: "keptn-api-token", Namespace: namespace}, keptnToken)

	secret := string(keptnToken.Data["keptn-api-token"])
	request, err := nethttp.NewRequest("POST", r.keptnApi+"/controlPlane/v1/project/"+project+"/service", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not create service "+service)
		return 0, err
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", secret)

	r.ReqLogger.Info("Creating Keptn Service " + service)
	response, err := httpclient.Do(request)
	if err != nil {
		return 0, err
	}
	return response.StatusCode, err
}

func (r *KeptnServiceReconciler) deleteService(service string, namespace string, project string) error {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	keptnToken := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: "keptn-api-token", Namespace: namespace}, keptnToken)

	secret := string(keptnToken.Data["keptn-api-token"])
	request, err := nethttp.NewRequest("DELETE", r.keptnApi+"/controlPlane/v1/project/"+project+"/service/"+service, bytes.NewBuffer(nil))
	if err != nil {
		r.ReqLogger.Error(err, "Could not delete service "+service)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", secret)

	r.ReqLogger.Info("Deleting Keptn Service " + service)
	_, err = httpclient.Do(request)
	if err != nil {
		return err
	}
	return err
}

func (r *KeptnServiceReconciler) triggerDeployment(service string, namespace string, project string, stage string, trigger string) error {

	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	data, err := json.Marshal(KeptnTriggerEvent{
		ContentType: "application/json",
		Data: KeptnEventData{
			Service: service,
			Project: project,
			Stage:   stage,
		},
		Source:      "Keptn GitOps Operator",
		SpecVersion: "1.0",
		Type:        trigger,
	})

	keptnToken := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: "keptn-api-token", Namespace: namespace}, keptnToken)

	secret := string(keptnToken.Data["keptn-api-token"])

	r.ReqLogger.Info("Triggering Deployment " + service)
	request, err := nethttp.NewRequest("POST", r.keptnApi+"/v1/event", bytes.NewBuffer(data))
	if err != nil {
		r.ReqLogger.Error(err, "Could not trigger deployment "+service)
		return err
	}

	request.Header.Set("content-type", "application/cloudevents+json")
	request.Header.Set("x-token", secret)

	_, err = httpclient.Do(request)
	if err != nil {
		return err
	}

	return err
}

func (r *KeptnServiceReconciler) checkKeptnServiceExists(service *keptnv1.KeptnService, namespace string) bool {
	httpclient := nethttp.Client{
		Timeout: 30 * time.Second,
	}

	keptnToken := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: "keptn-api-token", Namespace: namespace}, keptnToken)
	secret := string(keptnToken.Data["keptn-api-token"])

	request, err := nethttp.NewRequest("GET", r.keptnApi+"/configuration-service/v1/project/"+service.Spec.Project+"/stage/"+service.Spec.StartStage+"/service/"+service.Spec.Service+"/resource", bytes.NewBuffer(nil))
	request.Header.Set("x-token", secret)

	response, err := httpclient.Do(request)
	if err != nil || response.StatusCode != 200 {
		return false
	}
	r.ReqLogger.Info("Keptn Service already exists: " + service.Name)
	return true

}
