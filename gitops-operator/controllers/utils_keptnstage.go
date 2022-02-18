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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnstage/,verbs=get;list;create;update

func (r *KeptnGitRepositoryReconciler) checkCreateStage(ctx context.Context, repo gitopsv1.KeptnGitRepository, stage keptnv1.KeptnStage) (error, bool) {
	found := &keptnv1.KeptnStage{}

	stage.ObjectMeta.Namespace = repo.Namespace

	stage.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(stage.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &stage, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: stage.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Project", "Project.Namespace", repo.Namespace, "Project.Name", stage.Name)
		err = r.Client.Create(ctx, &stage)
		if err != nil {
			r.Log.Error(err, "Failed to create new Stage", "Stage.Namespace", repo.Namespace, "Stage.Name", stage.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get Stage")
		return err, false
	}

	err = r.reconcileStage(ctx, repo, stage)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileStage(ctx context.Context, repo gitopsv1.KeptnGitRepository, stage keptnv1.KeptnStage) error {
	obj := &keptnv1.KeptnStage{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: stage.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if stage.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = stage.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(stage.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Stage", "Stage.Namespace", obj.Namespace, "Stage.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated stage %s/%s (Reason: Stage changed)", stage.Namespace, stage.Name))
			r.Log.Info("Project updated")
		}
	}
	return nil
}
