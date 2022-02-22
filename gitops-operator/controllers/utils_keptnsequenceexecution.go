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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequenceexecutions,verbs=get;list;create;update;watch

func (r *KeptnGitRepositoryReconciler) checkCreateSequenceExecution(ctx context.Context, repo gitopsv1.KeptnGitRepository, sequenceExecution keptnv1.KeptnSequenceExecution) (error, bool) {
	found := &keptnv1.KeptnSequenceExecution{}

	sequenceExecution.ObjectMeta.Namespace = repo.Namespace

	sequenceExecution.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(sequenceExecution.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &sequenceExecution, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: sequenceExecution.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Project", "Project.Namespace", repo.Namespace, "Project.Name", sequenceExecution.Name)
		err = r.Client.Create(ctx, &sequenceExecution)
		if err != nil {
			r.Log.Error(err, "Failed to create new Project", "Project.Namespace", repo.Namespace, "Service.Name", sequenceExecution.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get Project")
		return err, false
	}

	err = r.reconcileSequenceExecution(ctx, repo, sequenceExecution)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileSequenceExecution(ctx context.Context, repo gitopsv1.KeptnGitRepository, sequenceExecution keptnv1.KeptnSequenceExecution) error {
	obj := &keptnv1.KeptnSequenceExecution{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: sequenceExecution.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if sequenceExecution.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = sequenceExecution.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(sequenceExecution.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Project", "Project.Namespace", obj.Namespace, "Project.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated project %s/%s (Reason: Project changed)", sequenceExecution.Namespace, sequenceExecution.Name))
			r.Log.Info("Project updated")
		}
	}
	return nil
}
