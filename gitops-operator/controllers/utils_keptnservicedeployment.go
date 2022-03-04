package controllers

import (
	"context"
	"fmt"
	gitopsv1 "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/api/v1"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservicedeployments,verbs=get;list;create;update;watch

func (r *KeptnGitRepositoryReconciler) checkCreateServiceDeployment(ctx context.Context, repo gitopsv1.KeptnGitRepository, serviceDeployment keptnv1.KeptnServiceDeployment) (error, bool) {
	found := &keptnv1.KeptnServiceDeployment{}

	serviceDeployment.ObjectMeta.Namespace = repo.Namespace

	serviceDeployment.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(serviceDeployment.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &serviceDeployment, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: serviceDeployment.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new KeptnServiceDeployment", "KeptnServiceDeployment.Namespace", repo.Namespace, "KeptnServiceDeployment.Name", serviceDeployment.Name)
		err = r.Client.Create(ctx, &serviceDeployment)
		if err != nil {
			r.Log.Error(err, "Failed to create new KeptnServiceDeployment", "KeptnServiceDeployment.Namespace", repo.Namespace, "KeptnServiceDeployment.Name", serviceDeployment.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get KeptnServiceDeployment")
		return err, false
	}

	err = r.reconcileServiceDeployment(ctx, repo, serviceDeployment)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileServiceDeployment(ctx context.Context, repo gitopsv1.KeptnGitRepository, serviceDeployment keptnv1.KeptnServiceDeployment) error {
	obj := &keptnv1.KeptnServiceDeployment{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: serviceDeployment.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if serviceDeployment.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = serviceDeployment.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(serviceDeployment.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update ServiceDeployment", "KeptnServiceDeployment.Namespace", obj.Namespace, "KeptnServiceDeployment.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated KeptnServiceDeployment %s/%s (Reason: KeptnServiceDeployment changed)", serviceDeployment.Namespace, serviceDeployment.Name))
			r.Log.Info("KeptnServiceDeployment updated")
		}
	}
	return nil
}
