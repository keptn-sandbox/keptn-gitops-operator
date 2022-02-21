package eventhandler

import (
	"errors"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	"github.com/keptn-sandbox/keptn-git-toolbox/promotion-service/common"
	"github.com/keptn-sandbox/keptn-git-toolbox/promotion-service/pkg/utils"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

const (
	promotionTaskName   = "promotion"
	defaultNamespace    = "keptn"
	namespaceEnvVarName = "POD_NAMESPACE"
	serviceName         = "promotion-service"
)

type PromotionHandler struct {
	Event        cloudevents.Event
	KeptnHandler *keptnv2.Keptn
	GitHandler   utils.GitHandlerInterface
}

// HandlePromotionTriggeredEvent handles promotion.triggered events
func (eh *PromotionHandler) HandlePromotionTriggeredEvent() error {
	eh.KeptnHandler.Logger.Info("Handling promotion.triggered event: " + eh.Event.Context.GetID())

	eventData := &keptnv2.EventData{}
	err := eh.Event.DataAs(eventData)
	if err != nil {
		eh.KeptnHandler.Logger.Error("Could not parse event payload: " + err.Error())
		return err
	}

	eh.KeptnHandler.Logger.Info("Sending promotion.started event")
	if err := eh.sendPromotionStartedEvent(); err != nil {
		eh.KeptnHandler.Logger.Error("Could not send promotion.started event: " + err.Error())
		return err
	}

	version, ok := eventData.Labels["version"]
	if !ok {
		err = errors.New("No version label given")
		eh.KeptnHandler.Logger.Error(err.Error())
		sendErr := eh.sendPromotionFinishedWithErrorEvent(err.Error())
		if sendErr != nil {
			eh.KeptnHandler.Logger.Error("Could not send promotion.finished with error event: " + sendErr.Error())
			return sendErr
		}
		return err
	} else {
		eh.KeptnHandler.Logger.Info("Using version: " + version)
	}

	namespaceSupplier := common.EnvBasedStringSupplier(namespaceEnvVarName, defaultNamespace)

	mysecret, err := utils.GetUpstreamCredentials(eventData.Project, namespaceSupplier())
	if err != nil {
		eh.KeptnHandler.Logger.Error(fmt.Sprintf("Could not fetch the secret for project %v: %v", eventData.Project, err.Error()))
		sendErr := eh.sendPromotionFinishedWithErrorEvent(err.Error())
		if sendErr != nil {
			eh.KeptnHandler.Logger.Error("Could not send promotion.finished with error event: " + sendErr.Error())
			return sendErr
		}
		return err
	}

	err = eh.GitHandler.UpdateGitRepo(mysecret, eventData.Stage, eventData.Service, version)
	if err != nil {
		count := 0
		for err != nil && count <= 5 {
			err = eh.GitHandler.UpdateGitRepo(mysecret, eventData.Stage, eventData.Service, version)
			count++
		}
		eh.KeptnHandler.Logger.Error(fmt.Sprintf("Could not update service %v/%v for stage %v: %v", eventData.Project, eventData.Service, eventData.Stage, err.Error()))
		sendErr := eh.sendPromotionFinishedWithErrorEvent(err.Error())
		if sendErr != nil {
			eh.KeptnHandler.Logger.Error("Could not send promotion.finished with error event: " + sendErr.Error())
		}
		return err
	}

	eh.KeptnHandler.Logger.Info("Sending promotion.finished event")
	if err := eh.sendPromotionFinishedWithSuccessEvent(); err != nil {
		eh.KeptnHandler.Logger.Error("Could not send promotion.finished event: " + err.Error())
		return err
	}

	return nil
}

func (eh *PromotionHandler) sendPromotionStartedEvent() error {
	eventData := keptnv2.EventData{
		Status: keptnv2.StatusSucceeded,
	}

	_, err := eh.KeptnHandler.SendTaskStartedEvent(&eventData, serviceName)
	return err
}

func (eh *PromotionHandler) sendPromotionFinishedWithSuccessEvent() error {
	eventData := keptnv2.EventData{
		Status: keptnv2.StatusSucceeded,
		Result: keptnv2.ResultPass,
	}

	_, err := eh.KeptnHandler.SendTaskFinishedEvent(&eventData, serviceName)
	return err
}

func (eh *PromotionHandler) sendPromotionFinishedWithErrorEvent(message string) error {
	eventData := keptnv2.EventData{
		Status:  keptnv2.StatusErrored,
		Result:  keptnv2.ResultFailed,
		Message: message,
	}

	_, err := eh.KeptnHandler.SendTaskFinishedEvent(&eventData, serviceName)
	return err
}
