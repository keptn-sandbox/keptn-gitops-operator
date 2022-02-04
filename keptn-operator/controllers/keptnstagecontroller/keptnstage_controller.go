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

package keptnstagecontroller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

// KeptnStageReconciler reconciles a KeptnStage object
type KeptnStageReconciler struct {
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

const reconcileErrorInterval = 10 * time.Second
const reconcileSuccessInterval = 120 * time.Second

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnstages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnstages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnstages/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnStage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *KeptnStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnStage")

	keptnstage := &apiv1.KeptnStage{}

	if err := r.Client.Get(ctx, req.NamespacedName, keptnstage); err != nil {
		if errors.IsNotFound(err) {
			// taking down all associated K8s resources is handled by K8s
			r.ReqLogger.Info("KeptnStage resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: true}, nil
		}
		r.ReqLogger.Error(err, "Failed to get the KeptnProject")
		return ctrl.Result{}, err
	}

	shipyard, err := utils.CreateShipyard(ctx, r.Client, keptnstage.Spec.Project)
	if err != nil {
		fmt.Println(err)
		r.ReqLogger.Error(err, "Could not create shipyard")
		return ctrl.Result{RequeueAfter: reconcileErrorInterval}, err
	}

	shipyardPresent, shipyardHash := utils.CheckKeptnShipyard(ctx, req, r.Client, keptnstage.Spec.Project)
	if !shipyardPresent {
		return ctrl.Result{RequeueAfter: reconcileErrorInterval, Requeue: true}, nil
	}

	err = utils.UpdateShipyard(ctx, r.Client, shipyard, shipyardHash, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not update shipyard")
		return ctrl.Result{RequeueAfter: reconcileErrorInterval, Requeue: true}, nil
	}

	r.ReqLogger.Info("Finished Reconciling KeptnStage")
	return ctrl.Result{RequeueAfter: reconcileSuccessInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnStageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.KeptnStage{}).
		Complete(r)
}
