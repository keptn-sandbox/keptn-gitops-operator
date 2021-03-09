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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"log"
	nethttp "net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keptnv1 "keptn-operator/api/v1"
)

// KeptnServiceReconciler reconciles a KeptnService object
type KeptnServiceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	keptnApi string
}

// +kubebuilder:rbac:groups=keptn.operator.keptn.sh,resources=keptnservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keptn.operator.keptn.sh,resources=keptnservices/status,verbs=get;update;patch

func (r *KeptnServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()

	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling KeptnService")

	var ok bool
	r.keptnApi, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		fmt.Println("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.keptnApi = "http://api-gateway-nginx/api"
	}

	service := &keptnv1.KeptnService{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, service)
	if errors.IsNotFound(err) {
		reqLogger.Info("KeptnProject resource not found. Ignoring since object must be deleted")
		return reconcile.Result{}, nil
	}

	service.Status.LastSetupStatus, err = r.createService(service.Spec.Service, req.Namespace, service.Spec.Project)
	if err != nil {
		fmt.Println("Service could not be created")
	}

	if service.Status.DeploymentPending {
		err = r.triggerDeployment(service.Spec.Service, req.Namespace, service.Spec.Project, service.Spec.StartStage, service.Spec.TriggerCommand)
		if err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}
		service.Status.DeploymentPending = false
	}

	if service.Status.DeletionPending {
		err = r.deleteService(service.Spec.Service, req.Namespace, service.Spec.Project)
		if err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}
		service.Status.SafeToDelete = true
	}

	err = r.Client.Update(context.TODO(), service)
	if err != nil {
		reqLogger.Error(err, "Could not update Service")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

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
		log.Fatalln(err)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", secret)

	log.Println("Creating Service in Keptn" + service)
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
		log.Fatalln(err)
	}

	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-token", secret)

	log.Println("Deleting Keptn Service " + service)
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

	log.Println("Triggering Deployment " + service)
	request, err := nethttp.NewRequest("POST", r.keptnApi+"/v1/event", bytes.NewBuffer(data))
	if err != nil {
		log.Fatalln(err)
	}

	request.Header.Set("content-type", "application/cloudevents+json")
	request.Header.Set("x-token", secret)

	response, err := httpclient.Do(request)

	fmt.Println(response)
	return err
}
