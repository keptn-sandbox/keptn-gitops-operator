package eventhandler

import (
	cloudevents "github.com/cloudevents/sdk-go/v2" // make sure to use v2 cloudevents here
	"github.com/cloudevents/sdk-go/v2/types"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
)

/*
func TestHandlePromotionTriggeredEvent(t *testing.T) {
	type channelEvent struct {
		Type string `json:"type"`
		Data struct {
			Status string `json:"status"`
			Result string `json:"result"`
		}
	}
	ch := make(chan *channelEvent)

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && strings.Contains(r.RequestURI, "/events") {
				defer r.Body.Close()
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Fatalf("Could not read body: %v", err)
				}
				fmt.Printf("Received request: %v\n", string(body))

				event := &channelEvent{}
				_ = json.Unmarshal(body, &event)

				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(200)
				_, err = w.Write([]byte(`{}`))
				if err != nil {
					log.Fatalf("Could not write response")
				}
				go func() { ch <- event }()
			}
		}),
	)
	defer ts.Close()

	os.Setenv("EVENTBROKER", ts.URL+"/events")

	////////// TEST DEFINITION ///////////
	type fields struct {
		Logger     *keptncommon.Logger
		Event      cloudevents.Event
		GitHandler utils.GitHandlerInterface
	}

	tests := []struct {
		name           string
		fields         fields
		wantEvents     []channelEvent
		wantErr        bool
		wantErrMessage string
	}{
		{
			name: "Promotion successful - send promotion.started and promotion.finished event",
			fields: fields{
				Logger: keptncommon.NewLogger("", "", ""),
				Event:  getPromotionTriggeredEvent(true),
				GitHandler: &githandler_mock.GitHandlerInterfaceMock{
					GetGitSecretFunc: func(project string, namespace string) (utils.GitRepositoryConfig, error) {
						return utils.GitRepositoryConfig{
							User:      "",
							Token:     "",
							RemoteURI: "",
						}, nil
					},
					UpdateGitRepoFunc: func(credentials utils.GitRepositoryConfig, stage string, service string, version string) error {
						return nil
					},
				},
			},
			wantEvents: []channelEvent{
				{
					Type: keptnv2.GetStartedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "succeeded",
					},
				},
				{
					Type: keptnv2.GetFinishedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "succeeded",
						Result: "pass",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Version label missing - send promotion.started and failed promotion.finished event",
			fields: fields{
				Logger: keptncommon.NewLogger("", "", ""),
				Event:  getPromotionTriggeredEvent(false),
			},
			wantEvents: []channelEvent{
				{
					Type: keptnv2.GetStartedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "succeeded",
					},
				},
				{
					Type: keptnv2.GetFinishedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "errored",
						Result: "fail",
					},
				},
			},
			wantErr:        true,
			wantErrMessage: "No version label given",
		},
		{
			name: "Git secret not fetchable - send promotion.started and failed promotion.finished event",
			fields: fields{
				Logger: keptncommon.NewLogger("", "", ""),
				Event:  getPromotionTriggeredEvent(true),
				GitHandler: &githandler_mock.GitHandlerInterfaceMock{
					GetGitSecretFunc: func(project string, namespace string) (utils.GitRepositoryConfig, error) {
						return utils.GitRepositoryConfig{}, errors.New("kubernetes secret error")
					},
				},
			},
			wantEvents: []channelEvent{
				{
					Type: keptnv2.GetStartedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "succeeded",
					},
				},
				{
					Type: keptnv2.GetFinishedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "errored",
						Result: "fail",
					},
				},
			},
			wantErr:        true,
			wantErrMessage: "kubernetes secret error",
		},
		{
			name: "Git secret not fetchable - send promotion.started and failed promotion.finished event",
			fields: fields{
				Logger: keptncommon.NewLogger("", "", ""),
				Event:  getPromotionTriggeredEvent(true),
				GitHandler: &githandler_mock.GitHandlerInterfaceMock{
					GetGitSecretFunc: func(project string, namespace string) (utils.GitRepositoryConfig, error) {
						return utils.GitRepositoryConfig{
							User:      "",
							Token:     "",
							RemoteURI: "",
						}, nil
					},
					UpdateGitRepoFunc: func(credentials utils.GitRepositoryConfig, stage string, service string, version string) error {
						return errors.New("git push error")
					},
				},
			},
			wantEvents: []channelEvent{
				{
					Type: keptnv2.GetStartedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "succeeded",
					},
				},
				{
					Type: keptnv2.GetFinishedEventType(promotionTaskName),
					Data: struct {
						Status string `json:"status"`
						Result string `json:"result"`
					}{
						Status: "errored",
						Result: "fail",
					},
				},
			},
			wantErr:        true,
			wantErrMessage: "git push error",
		},
	}

	////////// TEST EXECUTION ///////////
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			keptnHandler, _ := keptnv2.NewKeptn(&tt.fields.Event, keptncommon.KeptnOpts{
				EventBrokerURL: os.Getenv("EVENTBROKER"),
			})
			eh := &PromotionHandler{
				Event:        tt.fields.Event,
				KeptnHandler: keptnHandler,
				GitHandler:   tt.fields.GitHandler,
			}

			err := eh.HandlePromotionTriggeredEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("HandlePromotionTriggeredEvent() error = %v, wantErr %v, wantErrMessage %v", err, tt.wantErr, tt.wantErrMessage)
				return
			}
			if err != nil && err.Error() != tt.wantErrMessage {
				t.Errorf("HandlePromotionTriggeredEvent() error = %v, wantErr %v, wantErrMessage %v", err, tt.wantErr, tt.wantErrMessage)
				return
			}

			currentEventIndex := 0
			receivedExpected := 0
			var receivedEvents []*channelEvent
			for {
				select {
				case msg := <-ch:
					t.Logf("Received event: %+v", msg)
					receivedEvents = append(receivedEvents, msg)
					wantedEvent := tt.wantEvents[currentEventIndex]

					if msg.Type == wantedEvent.Type && msg.Data.Result == wantedEvent.Data.Result && msg.Data.Status == wantedEvent.Data.Status {
						receivedExpected = receivedExpected + 1
						currentEventIndex = currentEventIndex + 1
					}

					if receivedExpected == len(tt.wantEvents) {
						// received all events
						return
					}

				case <-time.After(5 * time.Second):
					t.Errorf("Expected messages did not make it to the receiver")
					t.Errorf("HandlePromotionTriggeredEvent() sent event type = %v, wantEvents %v", receivedEvents, tt.wantEvents)
					return
				}
			}
		})
	}
}


*/
func getPromotionTriggeredEvent(appendLabel bool) cloudevents.Event {
	data := `
    "project": "sockshop",
    "stage": "staging",
    "service": "carts",
    "promotion": null`

	if appendLabel {
		data = data + `,
    "labels": {
      "version": "1"
    }`
	}

	return cloudevents.Event{
		Context: &cloudevents.EventContextV1{
			Type:            keptnv2.GetTriggeredEventType(promotionTaskName),
			Source:          types.URIRef{},
			ID:              "",
			Time:            nil,
			DataContentType: stringp("application/json"),
			Extensions: map[string]interface{}{
				"shkeptncontext": "my-context",
			},
		},
		DataEncoded: []byte(`{
` + data + `
  }`),
		DataBase64: false,
	}
}

func stringp(s string) *string {
	return &s
}
