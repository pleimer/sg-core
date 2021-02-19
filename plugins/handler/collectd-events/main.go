package main

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/collectd-events/collectd"
)

type collectdEventsHandler struct{}

func (ch *collectdEventsHandler) Run(context.Context, bus.MetricPublishFunc, bus.EventPublishFunc) {

}

//Returns identification string for a handler
func (ch *collectdEventsHandler) Identify() string {
	return "collectd-events"
}

func (ch *collectdEventsHandler) Handle(blob []byte, reportErrs bool, mpf bus.MetricPublishFunc, epf bus.EventPublishFunc) error {
	event, err := collectd.Parse(blob)
	if err != nil {
		return err
	}
	epf(*event)
	return nil
}

func (ch *collectdEventsHandler) Config([]byte) error {
	return nil
}

//New handler constructor
func New() handler.Handler {
	return &collectdEventsHandler{}
}
