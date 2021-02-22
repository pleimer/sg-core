package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/events/pkg/lib"
)

//EventsHandler is processing event messages
type EventsHandler struct {
	eventsReceived map[string]uint64
	configuration  *lib.HandlerConfig
}

//ProcessingError contains processing error data
type ProcessingError struct {
	Error   string `json:"error"`
	Context string `json:"context"`
	Message string `json:"message"`
}

//Handle implements the data.EventsHandler interface
func (eh *EventsHandler) Handle(msg []byte, reportErrors bool, sendMetric bus.MetricPublishFunc, sendEvent bus.EventPublishFunc) error {
	source := lib.DataSource(0)
	if eh.configuration.StrictSource != "" {
		source.SetFromString(eh.configuration.StrictSource)
	} else {
		// if strict source is not set then handler is processing channel with multiple data sources
		// and has to be recognized from message format
		source.SetFromMessage(msg)
	}

	if _, ok := eh.eventsReceived[source.String()]; !ok {
		eh.eventsReceived[source.String()] = uint64(0)
	}
	eh.eventsReceived[source.String()]++

	// sanitize received message based on data source
	//TODO: refactor sanitizers to avoid string conversion
	sanitized := []byte(lib.Sanitizers[source.String()](msg))
	// format data based on data source
	var message map[string]interface{}
	err := json.Unmarshal(sanitized, &message)
	if err != nil {
		if reportErrors {
			sendEvent(eh.Identify(), data.ERROR, map[string]interface{}{
				"error":   err.Error(),
				"context": string(msg),
				"message": "failed to unmarshal event - disregarding",
			})
		}
		return err
	}
	// format message if needed
	err = lib.EventFormatters[source.String()](message)
	if err != nil {
		if reportErrors {
			sendEvent(eh.Identify(), data.ERROR, map[string]interface{}{
				"error":   err.Error(),
				"context": fmt.Sprintf("%v", message),
				"message": "failed to format event - disregarding",
			})
		}
		return err
	}
	// wrap event message with necessary metadata and marshal the structure
	event := map[string]interface{}{
		"source":  source.String(),
		"message": message,
	}
	// send internal event
	sendEvent(eh.Identify(), data.EVENT, event)
	return nil
}

//Run send internal metrics to bus
func (eh *EventsHandler) Run(ctx context.Context, sendMetric bus.MetricPublishFunc, sendEvent bus.EventPublishFunc) {
	for {
		select {
		case <-ctx.Done():
			goto done
		case <-time.After(time.Second):
			total := uint64(0)
			for source, value := range eh.eventsReceived {
				sendMetric(
					fmt.Sprintf("sg_%s_events_received", source),
					0,
					data.COUNTER,
					0,
					float64(value),
					[]string{"source"},
					[]string{"SG"},
				)
				total += value
			}
			sendMetric(
				"sg_total_events_received",
				0,
				data.COUNTER,
				0,
				float64(total),
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

//Config ...
func (eh *EventsHandler) Config(blob []byte) error {
	eh.configuration = &lib.HandlerConfig{StrictSource: ""}
	return config.ParseConfig(bytes.NewReader(blob), eh.configuration)
}

//New create new eventsHandler object
func New() handler.Handler {
	return &EventsHandler{eventsReceived: make(map[string]uint64)}
}
