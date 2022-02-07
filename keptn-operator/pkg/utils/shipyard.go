package utils

import (
	"context"
	"fmt"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const shipyardAPIVersion = "spec.keptn.sh/0.2.2"
const shipyardKind = "KeptnShipyard"

//UpdateShipyard triggers the update of the shipyard object
func UpdateShipyard(ctx context.Context, clt client.Client, shipyard keptnv1.KeptnShipyard, shipyardHash string, namespace string) error {
	shipyard.Namespace = namespace
	shipyard.Status.LastAppliedHash = GetHashStructure(shipyard.Spec.Shipyard)

	if GetHashStructure(shipyard.Spec.Shipyard) != shipyardHash {
		currentShipyard := &keptnv1.KeptnShipyard{}
		err := clt.Get(ctx, types.NamespacedName{Name: shipyard.Name, Namespace: namespace}, currentShipyard)
		if err != nil {
			return fmt.Errorf("could not get shipyard: %w", err)
		}

		currentShipyard.Spec = shipyard.Spec
		currentShipyard.Status.LastAppliedHash = GetHashStructure(currentShipyard.Spec)
		err = clt.Update(ctx, currentShipyard)
		if err != nil {
			return fmt.Errorf("could not update status of shipyard: %w", err)
		}
	}
	return nil
}

//CreateShipyard creates a shipyard object
func CreateShipyard(ctx context.Context, clt client.Client, project string) (keptnv1.KeptnShipyard, error) {
	shipyard := keptnv1.KeptnShipyard{}

	shipyard.Name = project
	shipyard.Spec.Project = project

	keptnShipyard, err := composeShipyard(ctx, clt, project)
	if err != nil {
		return shipyard, err
	}

	shipyard.Spec.Shipyard = keptnShipyard
	if err != nil {
		return shipyard, err
	}

	return shipyard, nil
}

func composeShipyard(ctx context.Context, clt client.Client, project string) (keptnv1.Shipyard, error) {
	keptnShipyard := keptnv1.Shipyard{
		ApiVersion: shipyardAPIVersion,
		Kind:       shipyardKind,
		Metadata:   keptnv1.Metadata{},
		Spec:       keptnv1.ShipyardSpec{},
	}

	stages, err := getKeptnStages(ctx, clt, project)
	if err != nil {
		return keptnShipyard, err
	}

	sequences, err := getKeptnSequence(ctx, clt)
	if err != nil {
		return keptnShipyard, err
	}

	for _, stg := range stages {
		stage, err := composeKeptnStage(stg, sequences)
		if err != nil {
			return keptnShipyard, err
		}
		keptnShipyard.Spec.Stages = append(keptnShipyard.Spec.Stages, stage)
		fmt.Println(keptnShipyard.Spec.Stages)

	}
	return keptnShipyard, nil
}

func composeKeptnStage(stage keptnv1.KeptnStage, sequences *keptnv1.KeptnSequenceList) (keptnv1.Stage, error) {
	compstage := keptnv1.Stage{
		Name: stage.Name,
	}

	for _, seq := range stage.Spec.Sequence {
		sequenceFound := false
		for _, availableSequence := range sequences.Items {
			if availableSequence.Name == seq.SequenceRef {
				sequenceFound = true

				sequence := availableSequence.Spec.Sequence
				compstage.Sequences = append(compstage.Sequences, sequence)
				break
			}
		}
		if !sequenceFound {
			return compstage, fmt.Errorf("could not find sequence")
		}
	}
	return compstage, nil
}

func getKeptnStages(ctx context.Context, clt client.Client, project string) ([]keptnv1.KeptnStage, error) {
	keptnStageList := &keptnv1.KeptnStageList{}
	var stageList []keptnv1.KeptnStage

	err := clt.List(ctx, keptnStageList)
	if err != nil {
		return stageList, fmt.Errorf("could not get stages for project: %w", err)
	}

	for _, stage := range keptnStageList.Items {
		if stage.Spec.Project == project {
			stageList = append(stageList, stage)
		}
	}

	return stageList, nil
}

func getKeptnSequence(ctx context.Context, clt client.Client) (*keptnv1.KeptnSequenceList, error) {
	sequenceList := &keptnv1.KeptnSequenceList{}
	err := clt.List(ctx, sequenceList)
	if err != nil {
		return sequenceList, fmt.Errorf("could not get sequences for project: %w", err)
	}
	return sequenceList, nil
}

//CheckKeptnShipyard checks if a keptn shipyard object exists
func CheckKeptnShipyard(ctx context.Context, req ctrl.Request, client client.Client, project string) (bool, string) {
	shipyardRes := &keptnv1.KeptnShipyard{}

	err := client.Get(ctx, types.NamespacedName{Name: project, Namespace: req.Namespace}, shipyardRes)
	if err != nil {
		return false, ""
	}
	return true, shipyardRes.Status.LastAppliedHash
}
