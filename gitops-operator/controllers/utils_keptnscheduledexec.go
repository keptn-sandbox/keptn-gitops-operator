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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnscheduledexecutions,verbs=get;list;create;update;watch

func (r *KeptnGitRepositoryReconciler) checkCreateScheduledExecution(ctx context.Context, repo gitopsv1.KeptnGitRepository, scheduledExecution keptnv1.KeptnScheduledExec) (error, bool) {
	found := &keptnv1.KeptnScheduledExec{}

	scheduledExecution.ObjectMeta.Namespace = repo.Namespace

	scheduledExecution.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(scheduledExecution.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &scheduledExecution, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: scheduledExecution.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Scheduled Execution", "ScheduledExec.Namespace", repo.Namespace, "ScheduledExec.Name", scheduledExecution.Name)
		err = r.Client.Create(ctx, &scheduledExecution)
		if err != nil {
			r.Log.Error(err, "Failed to create new Project", "ScheduledExec.Namespace", repo.Namespace, "ScheduledExec.Name", scheduledExecution.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get Project")
		return err, false
	}

	err = r.reconcileScheduledExecution(ctx, repo, scheduledExecution)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileScheduledExecution(ctx context.Context, repo gitopsv1.KeptnGitRepository, scheduledExecution keptnv1.KeptnScheduledExec) error {
	obj := &keptnv1.KeptnScheduledExec{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: scheduledExecution.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if scheduledExecution.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = scheduledExecution.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(scheduledExecution.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Project", "Sequence.Namespace", obj.Namespace, "Sequence.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated scheduledExecution %s/%s (Reason: scheduledExecution changed)", scheduledExecution.Namespace, scheduledExecution.Name))
			r.Log.Info("Project updated")
		}
	}
	return nil
}
