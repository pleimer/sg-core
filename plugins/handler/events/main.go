package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/events/pkg/lib"
)

//EventsHandler is processing event messages
type EventsHandler struct {
	totalEventsReceived uint64
}

//ProcessedEvent contains event data with additional context metadata
type ProcessedEvent struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

//ProcessingError contains processing error data
type ProcessingError struct {
	Error   string `json:"error"`
	Context string `json:"context"`
	Message string `json:"message"`
}

//Handle implements the data.EventsHandler interface
func (eh *EventsHandler) Handle(msg []byte, reportErrors bool, sendMetric bus.MetricPublishFunc, sendEvent bus.EventPublishFunc) error {
	eh.totalEventsReceived++

	source := lib.DataSource(0)
	source.SetFromMessage(msg)
	// sanitize received message based on data source
	//TODO: refactor sanitizers to avoid string conversion
	sanitized := []byte(lib.Sanitizers[source.String()](msg))
	// format data based on data source
	var message map[string]interface{}
	err := json.Unmarshal(sanitized, &message)
	if err != nil {
		if reportErrors {
			errmsg, errr := json.Marshal(ProcessingError{
				Error:   err.Error(),
				Context: string(msg),
				Message: "failed to unmarshal event - disregarding",
			})
			if errr != nil {
				sendEvent(eh.Identify(), data.ERROR, errmsg)
			}
		}
		return err
	}
	// format message if needed
	err = lib.EventFormatters[source.String()](message)
	if err != nil {
		if reportErrors {
			errmsg, err := json.Marshal(ProcessingError{
				Error:   err.Error(),
				Context: fmt.Sprintf("%v", message),
				Message: "failed to format event - disregarding",
			})
			if err != nil {
				sendEvent(eh.Identify(), data.ERROR, errmsg)
			}
		}
		return err
	}
	// wrap event message with necessary metadata and marshal the structure
	event := map[string]interface{}{
		"source":  source.String(),
		"message": message,
	}
	evt, err := json.Marshal(event)
	if err != nil {
		if reportErrors {
			errmsg, err := json.Marshal(ProcessingError{
				Error:   err.Error(),
				Context: fmt.Sprintf("%v", event),
				Message: "failed to format event - disregarding",
			})
			if err != nil {
				sendEvent(eh.Identify(), data.ERROR, errmsg)
			}
		}
		return err
	}
	// send internal event
	sendEvent(eh.Identify(), data.EVENT, evt)
	return nil
}

//Run send internal metrics to bus
func (eh *EventsHandler) Run(ctx context.Context, sendMetric bus.MetricPublishFunc, sendEvent bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second):
			sendMetric(
				"sg_total_events_received",
				0,
				data.COUNTER,
				0,
				float64(eh.totalEventsReceived),
				[]string{"source"},
				[]string{"SG"},
			)
		}
	}
done:
}

//Identify returns handler's name
func (eh *EventsHandler) Identify() string {
	return "events"
}

//New create new collectdEventsHandler object
func New() handler.Handler {
	return &EventsHandler{0}
}
