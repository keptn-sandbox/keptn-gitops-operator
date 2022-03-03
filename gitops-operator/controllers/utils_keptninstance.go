package controllers

import (
	"context"
	"fmt"
	gitopsv1 "github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator/api/v1"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

//+kubebuilder:rbac:groups=keptn.sh,resources=keptninstances,verbs=get;list;create;update;watch

func (r *KeptnGitRepositoryReconciler) checkCreateInstance(ctx context.Context, repo gitopsv1.KeptnGitRepository, instance keptnv1.KeptnInstance) (bool, error) {
	found := &keptnv1.KeptnInstance{}

	instance.ObjectMeta.Namespace = repo.Namespace

	instance.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(instance.Spec),
	}

	err := r.Client.Get(ctx, types.NamespacedName{Name: instance.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Instance", "Instance.Namespace", repo.Namespace, "Instance.Name", instance.Name)
		err = r.Client.Create(ctx, &instance)
		if err != nil {
			r.Log.Error(err, "Failed to create new Instance", "Instance.Namespace", repo.Namespace, "Instance.Name", instance.Name)
			return false, err
		}
		return true, nil
	} else if err != nil {
		r.Log.Error(err, "Failed to get Instance")
		return false, err
	}

	err = r.reconcileInstance(ctx, repo, instance)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (r *KeptnGitRepositoryReconciler) reconcileInstance(ctx context.Context, repo gitopsv1.KeptnGitRepository, instance keptnv1.KeptnInstance) error {
	obj := &keptnv1.KeptnInstance{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: instance.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if instance.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = instance.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(instance.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Instance", "Instance.Namespace", obj.Namespace, "Instance.Name", obj.Name)
			return err
		}
		r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated instance %s/%s (Reason: Instance changed)", instance.Namespace, instance.Name))
		r.Log.Info("Project updated")

	}
	return nil
}
