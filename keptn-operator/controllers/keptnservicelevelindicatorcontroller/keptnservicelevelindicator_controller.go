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

package keptnservicelevelindicatorcontroller

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"

	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	// "github.com/keptn/go-utils/pkg/api/models"
	// apiutils "github.com/keptn/go-utils/pkg/api/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keptnshv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
)

// KeptnServiceLevelIndicatorReconciler reconciles a KeptnServiceLevelIndicator object
type KeptnServiceLevelIndicatorReconciler struct {
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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicelevelindicators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicelevelindicators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicelevelindicators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KeptnServiceLevelIndicator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *KeptnServiceLevelIndicatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ReqLogger = ctrl.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	r.ReqLogger.Info("Reconciling KeptnServiceLevelIndicator")

	var err error
	r.KeptnInstance, r.KeptnAPIToken, err = utils.GetKeptnInstance(ctx, r.Client, req.Namespace)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get Keptn Instance")
		return ctrl.Result{Requeue: true, RequeueAfter: reconcileErrorInterval}, nil
	}

	ksli := &apiv1.KeptnServiceLevelIndicator{}

	if ksli.Spec.Project == "" {
		r.ReqLogger.Error(err, "Could not apply service level indicators as the project is not specified")
		return ctrl.Result{}, nil
	}

	// resourceHandler := apiutils.NewAuthenticatedResourceHandler(r.KeptnInstance.Spec.APIUrl, r.KeptnAPIToken, r.KeptnInstance.Status.AuthHeader, nil, r.KeptnInstance.Status.Scheme)

	if ksli.Spec.Service != "" {
		if ksli.Spec.Stage == "" {
			r.ReqLogger.Error(err, "Could not apply service level indicators to the %s service as the stage is not specified", ksli.Spec.Service)
			return ctrl.Result{}, nil
		}
	} else if ksli.Spec.Stage != "" {
		// apply sli to all services of stage but NOT to rest of project
	}

	// apply sli to all services of all stages

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KeptnServiceLevelIndicatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keptnshv1.KeptnServiceLevelIndicator{}).
		Complete(r)
}
