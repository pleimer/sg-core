package main

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/handler"
	"github.com/infrawatch/sg-core/plugins/handler/ceilometer-events/ceilometer"
)

type ceilometerEvents struct {
	totalEventsReceived uint64
}

//Run main process
func (ce *ceilometerEvents) Run(context.Context, bus.MetricPublishFunc, bus.EventPublishFunc) {

}

//Identify return ID
func (ce *ceilometerEvents) Identify() string {
	return "ceilometer-events"
}

//Handle handle incoming event messages
func (ce *ceilometerEvents) Handle(blob []byte, done bool, mrf bus.MetricPublishFunc, ewf bus.EventPublishFunc) error {
	ce.totalEventsReceived++

	ceilometer.Parse(blob)

	return nil
}

//Config config
func (ce *ceilometerEvents) Config([]byte) error {
	return nil
}

//New constructor for sg-core to invoke this plugin with
func New() handler.Handler {
	return &ceilometerEvents{}
}
