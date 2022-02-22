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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnservices,verbs=get;list;create;update;watch

func (r *KeptnGitRepositoryReconciler) checkCreateService(ctx context.Context, repo gitopsv1.KeptnGitRepository, service keptnv1.KeptnService) (error, bool) {
	found := &keptnv1.KeptnService{}

	service.ObjectMeta.Namespace = repo.Namespace

	service.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(service.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &service, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: service.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Service", "Service.Namespace", repo.Namespace, "Service.Name", service.Name)
		err = r.Client.Create(ctx, &service)
		if err != nil {
			r.Log.Error(err, "Failed to create new Service", "Service.Namespace", repo.Namespace, "Service.Name", service.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get Service")
		return err, false
	}

	err = r.reconcileService(ctx, repo, service)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileService(ctx context.Context, repo gitopsv1.KeptnGitRepository, service keptnv1.KeptnService) error {
	obj := &keptnv1.KeptnService{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: service.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if service.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = service.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(service.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Service", "Service.Namespace", obj.Namespace, "Service.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated service %s/%s (Reason: Service changed)", service.Namespace, service.Name))
			r.Log.Info("Service updated")
		}
	}
	return nil
}
