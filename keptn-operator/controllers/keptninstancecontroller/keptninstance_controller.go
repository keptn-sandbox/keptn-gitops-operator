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

package keptninstancecontroller

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
)

// KeptnInstanceReconciler reconciles a KeptnInstance object
type KeptnInstanceReconciler struct {
	client.Client

	// Scheme contains the scheme of this controller
	Scheme *runtime.Scheme
	// Recorder contains the Recorder of this controller
	Recorder record.EventRecorder
	// ReqLogger contains the Logger of this controller
	ReqLogger logr.Logger
}

const reconcileErrorInterval = 10 * time.Second
const reconcileSuccessInterval = 120 * time.Second
const refreshInterval = 120 * time.Second

//+kubebuilder:rbac:groups=keptn.sh,resources=keptninstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptninstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptninstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnSequenceExecution")

	instance := &apiv1.KeptnInstance{}

	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnInstance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnInstance")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
	}

	url, err := url.Parse(instance.Spec.APIUrl)
	if err != nil {
		panic(err)
	}
	instance.Status.Scheme = url.Scheme

	switch instance.Spec.TokenType {
	case "internal":
		token, err := utils.GetKeptnCPToken(ctx, r.Client, req.Namespace)
		if err != nil {
			r.ReqLogger.Error(err, "Could not get Keptn Token")
		}
		encToken, err := utils.EncryptPublicPEM(token)
		if err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		instance.Status.AuthHeader = "x-token"
		instance.Status.CurrentToken = encToken

		if encToken != instance.Status.CurrentToken || instance.Status.LastUpdated.Add(refreshInterval).Before(time.Now()) {
			instance.Status.LastUpdated = metav1.Time{Time: time.Now()}
			err = r.Client.Status().Update(ctx, instance)
			if err != nil {
				r.ReqLogger.Error(err, "Could not update status of keptninstance "+instance.Name)
				return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
			}
			return ctrl.Result{Requeue: true}, err
		}
	case "x-token":
		instance.Status.AuthHeader = "x-token"
		instance.Status.CurrentToken = instance.Spec.Token

		if instance.Spec.Token != instance.Status.CurrentToken || instance.Status.LastUpdated.Add(refreshInterval).Before(time.Now()) {
			instance.Status.LastUpdated = metav1.Time{Time: time.Now()}
			err = r.Client.Status().Update(ctx, instance)
			if err != nil {
				r.ReqLogger.Error(err, "Could not update status of keptninstance "+instance.Name)
				return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, err
			}
			return ctrl.Result{Requeue: true}, err
		}

		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, err
	}

	r.ReqLogger.Info("Finished Reconciling KeptnInstance")
	return ctrl.Result{RequeueAfter: reconcileSuccessInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnInstance{}).
		Complete(r)
}
