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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnsequences/,verbs=get;list;create;update

func (r *KeptnGitRepositoryReconciler) checkCreateSequence(ctx context.Context, repo gitopsv1.KeptnGitRepository, sequence keptnv1.KeptnSequence) (error, bool) {
	found := &keptnv1.KeptnSequence{}

	sequence.ObjectMeta.Namespace = repo.Namespace

	sequence.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(sequence.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &sequence, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: sequence.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Project", "Project.Namespace", repo.Namespace, "Project.Name", sequence.Name)
		err = r.Client.Create(ctx, &sequence)
		if err != nil {
			r.Log.Error(err, "Failed to create new Project", "Project.Namespace", repo.Namespace, "Service.Name", sequence.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get Project")
		return err, false
	}

	err = r.reconcileSequence(ctx, repo, sequence)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileSequence(ctx context.Context, repo gitopsv1.KeptnGitRepository, sequence keptnv1.KeptnSequence) error {
	obj := &keptnv1.KeptnSequence{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: sequence.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if sequence.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = sequence.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(sequence.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Project", "Sequence.Namespace", obj.Namespace, "Sequence.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated sequence %s/%s (Reason: Project changed)", sequence.Namespace, sequence.Name))
			r.Log.Info("Project updated")
		}
	}
	return nil
}
