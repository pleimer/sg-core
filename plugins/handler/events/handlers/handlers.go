package handlers

import (
	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/plugins/handler/events/ceilometer"
	"github.com/infrawatch/sg-core/plugins/handler/events/collectd"
)

func ceilometerEventHandler(blob []byte, epf bus.EventPublishFunc) error {
	ceilo := ceilometer.Ceilometer{}

	err := ceilo.Parse(blob)
	if err != nil {
		return err
	}

	err = ceilo.IterEvents(func(e data.Event) {
		epf(e)
	})

	if err != nil {
		return err
	}
	return nil
}
func collectdEventHandler(blob []byte, epf bus.EventPublishFunc) error {
	event, err := collectd.Parse(blob)
	if err != nil {
		return err
	}
	epf(*event)
	return nil
}

//EventHandlers handle messages according to the expected data source and write parsed events to the events bus
var EventHandlers = map[string]func([]byte, bus.EventPublishFunc) error{
	"ceilometer": ceilometerEventHandler,
	"collectd":   collectdEventHandler,
}
