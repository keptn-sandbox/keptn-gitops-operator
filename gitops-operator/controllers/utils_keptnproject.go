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

//+kubebuilder:rbac:groups=keptn.sh,resources=keptnprojects,verbs=get;list;create;update;watch

func (r *KeptnGitRepositoryReconciler) checkCreateProject(ctx context.Context, repo gitopsv1.KeptnGitRepository, project keptnv1.KeptnProject) (error, bool) {
	found := &keptnv1.KeptnProject{}

	project.ObjectMeta.Namespace = repo.Namespace

	project.ObjectMeta.Annotations = map[string]string{
		"keptn.sh/last-applied-hash": utils.GetHashStructure(project.Spec),
	}

	err := controllerutil.SetControllerReference(&repo, &project, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not set controller reference: %w", err), false
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: project.ObjectMeta.Name, Namespace: repo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Project", "Project.Namespace", repo.Namespace, "Project.Name", project.Name)
		err = r.Client.Create(ctx, &project)
		if err != nil {
			r.Log.Error(err, "Failed to create new Project", "Project.Namespace", repo.Namespace, "Project.Name", project.Name)
			return err, false
		}
		return nil, true
	} else if err != nil {
		r.Log.Error(err, "Failed to get Project")
		return err, false
	}

	err = r.reconcileProject(ctx, repo, project)
	if err != nil {
		return err, false
	}

	return nil, false
}

func (r *KeptnGitRepositoryReconciler) reconcileProject(ctx context.Context, repo gitopsv1.KeptnGitRepository, project keptnv1.KeptnProject) error {
	obj := &keptnv1.KeptnProject{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: project.Name, Namespace: repo.Namespace}, obj)
	if err != nil {
		return err
	}

	if project.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] != obj.Annotations["keptn.sh/last-applied-hash"] {
		obj.Spec = project.Spec
		obj.ObjectMeta.Annotations["keptn.sh/last-applied-hash"] = utils.GetHashStructure(project.Spec)

		err := r.Client.Update(ctx, obj)
		if err != nil {
			r.Log.Error(err, "Failed to update Project", "Project.Namespace", obj.Namespace, "Project.Name", obj.Name)
			return err
		} else {
			r.Recorder.Event(&repo, "Normal", "Updated", fmt.Sprintf("Updated project %s/%s (Reason: Project changed)", project.Namespace, project.Name))
			r.Log.Info("Project updated")
		}
	}
	return nil
}
