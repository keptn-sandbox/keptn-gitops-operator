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

package keptnscheduledexeccontroller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KeptnScheduledExecReconciler reconciles a KeptnScheduledExec object
type KeptnScheduledExecReconciler struct {
	client.Client

	// Scheme contains the scheme of this controller
	Scheme *runtime.Scheme
	// Recorder contains the Recorder of this controller
	Recorder record.EventRecorder
	// ReqLogger contains the Logger of this controller
	ReqLogger logr.Logger
	// KeptnAPI contains the URL of the Keptn Control Plane API
	KeptnAPI string
	// KeptnAPIScheme contains the Scheme (http/https) of the Keptn Control Plane API
	KeptnAPIScheme string
}

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnscheduledexecs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnscheduledexecs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnscheduledexecs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnScheduledExec object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnScheduledExecReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnScheduledExec")

	var ok bool
	r.KeptnAPI, ok = os.LookupEnv("KEPTN_API_ENDPOINT")
	if !ok {
		r.ReqLogger.Info("KEPTN_API_ENDPOINT is not present, defaulting to api-gateway-nginx")
		r.KeptnAPI = "http://api-gateway-nginx/api"
	}

	if r.KeptnAPIScheme == "" {
		r.KeptnAPIScheme = "http"
	}

	keptnexec := &apiv1.KeptnScheduledExec{}

	if err := r.Client.Get(ctx, req.NamespacedName, keptnexec); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnScheduledExec resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnScheduledExec")
		return ctrl.Result{}, err
	}

	scheduledTime, err := time.Parse(time.RFC3339, keptnexec.Spec.StartTime)
	if err != nil {
		return ctrl.Result{}, err
	}

	if scheduledTime.After(time.Now()) && keptnexec.Status.Started == true {
		keptnexec.Status.Started = false
		err = r.Client.Status().Update(ctx, keptnexec)
		if err != nil {
			fmt.Println(err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if scheduledTime.Before(time.Now()) && keptnexec.Status.Started == false {
		fmt.Println("Would trigger execution")
		seq := apiv1.KeptnSequenceExecution{
			ObjectMeta: v1.ObjectMeta{
				GenerateName: string("scheduledexecution-"),
				Namespace:    req.Namespace,
			},
			Spec: keptnexec.Spec.SequenceExecutionTemplate,
		}

		err := r.Client.Create(ctx, &seq)
		if err != nil {
			fmt.Println(err)
		}

		keptnexec.Status.Started = true
		err = r.Client.Status().Update(ctx, keptnexec)
		if err != nil {
			fmt.Println(err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	r.ReqLogger.Info("Finished Reconciling KeptnScheduledExec")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnScheduledExecReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnScheduledExec{}).
		Complete(r)
}
