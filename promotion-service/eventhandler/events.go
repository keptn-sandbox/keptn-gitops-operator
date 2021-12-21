package eventhandler

import "github.com/keptn/go-utils/pkg/lib/v0_2_0"

// EchoEventTriggeredType is the name of an echo triggered event
const PromotionEventTriggeredType = "sh.keptn.event.promotion.triggered"

// EchoStartedEventType is the name of an echo started event
const PromotionStartedEventType = "sh.keptn.event.promotion.started"

// EchoFinishedEventType is the name of an echo finished event
const PromotionFinishedEventType = "sh.keptn.event.promotion.finished"

// EchoTriggeredEventData is the data of an echo triggered event
type PromotionTriggeredEventData struct {
	v0_2_0.EventData
}

// EchoStartedEventData is the data of an echo started event
type PromotionStartedEventData struct {
	v0_2_0.EventData
}

// EchoFinishedEventData is the data of an echo finished event
type PromotionFinishedEventData struct {
	v0_2_0.EventData
}
