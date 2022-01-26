package keptnservicedeploymentcontroller

// KeptnTriggerEvent describes a Keptn Event which should be triggered
type KeptnTriggerEvent struct {
	ContentType string         `json:"contenttype,omitempty"`
	Data        KeptnEventData `json:"data,omitempty"`
	Source      string         `json:"source,omitempty"`
	SpecVersion string         `json:"specversion,omitempty"`
	Type        string         `json:"type,omitempty"`
	Context     string         `json:"shkeptncontext,omitempty"`
}

// KeptnEventData describes the Event Data of an KeptnTriggerEvent
type KeptnEventData struct {
	Project             string                  `json:"project,omitempty"`
	Service             string                  `json:"service,omitempty"`
	Stage               string                  `json:"stage,omitempty"`
	Image               string                  `json:"image,omitempty"`
	Labels              map[string]string       `json:"labels,omitempty"`
	ConfigurationChange ConfigurationChangeData `json:"configurationChange,omitempty"`
}

// ConfigurationChangeData describes the Configuration Change block of a KeptnEventData
type ConfigurationChangeData struct {
	Values map[string]string `json:"values,omitempty"`
}

// CreateEventResponse describes the Response of a sent Keptn Event
type CreateEventResponse struct {
	KeptnContext string `json:"keptnContext"`
}
