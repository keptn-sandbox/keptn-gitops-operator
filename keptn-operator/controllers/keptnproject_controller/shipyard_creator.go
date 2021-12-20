package keptnproject_controller

import (
	"context"
	apiv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KeptnProjectReconciler) createShipyard(ctx context.Context, project *apiv1.KeptnProject) apiv1.KeptnShipyard {
	shipyard := apiv1.KeptnShipyard{}

	shipyard.Name = project.Name
	shipyard.Spec.Project = project.Name

	for _, stage := range r.getKeptnStages(ctx, project.Name) {
		shipyard.Spec.Stages = append(shipyard.Spec.Stages, apiv1.KeptnShipyardStage{StageRef: stage.Name})
	}

	return shipyard
}

func (r *KeptnProjectReconciler) getKeptnStages(ctx context.Context, project string) []apiv1.KeptnStage {
	var keptnStageList apiv1.KeptnStageList
	var stageList []apiv1.KeptnStage

	listOpts := []client.ListOption{}

	err := r.Client.List(ctx, &keptnStageList, listOpts...)
	if err != nil {
		r.ReqLogger.Error(err, "Could not get keptn stages")
		return stageList
	}

	for _, stage := range keptnStageList.Items {
		if stage.Spec.Project == project {
			stageList = append(stageList, stage)
		}
	}
	return stageList
}
