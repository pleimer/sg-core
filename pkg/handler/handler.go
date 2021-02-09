package handler

import (
	"context"

	"github.com/infrawatch/sg-core/pkg/bus"
)

// package handler contains the interface description for handler plugins

//Handler mangle messages to place on metric bus
type Handler interface {
	//Run should only be used to send metrics or events apart from those being parsed from the transport. For example, this process could send metrics tracking the number of arrived messages and send them to the bus on a time delayed interval
	Run(context.Context, bus.MetricPublishFunc, bus.EventPublishFunc)

	//Returns identification string for a handler
	Identify() string

	//Handle parse incoming messages from the transport and write resulting metrics or events to the corresponding bus
	Handle([]byte, bool, bus.MetricPublishFunc, bus.EventPublishFunc) error
}
