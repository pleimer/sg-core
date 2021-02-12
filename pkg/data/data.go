package data

import (
	"fmt"
	"time"
)

// package data defines the data descriptions for objects used in the internal buses

// MetricType follows standard metric conventions from prometheus
type MetricType int

const (
	//COUNTER ...
	COUNTER MetricType = iota
	//GAUGE ...
	GAUGE
	//UNTYPED ...
	UNTYPED
)

// EventType marks type of data held in event message
type EventType int

const (
	// ERROR event contains handler failure data and should be handled on application level
	ERROR EventType = iota
	// EVENT contains regular event data
	EVENT
	// RESULT event contains data about result of check execution
	// perfomed by any supported client side agent (collectd-sensubility, sg-agent)
	RESULT
	// LOG event contains log record
	LOG
)

func (et EventType) String() string {
	return []string{"error", "event", "result", "log"}[et]
}

// Event internal event type
type Event struct {
	Handler string
	Type    EventType
	Message string
}

// Metric internal metric type
type Metric struct {
	Name      string
	Labels    map[string]string
	LabelKeys []string
	LabelVals []string
	Time      float64
	Type      MetricType
	Interval  time.Duration
	Value     float64
}

//DataSource indentifies a format of incoming data in the message bus channel.
type DataSource int

//ListAll returns slice of supported data sources in human readable names.
func (src DataSource) ListAll() []string {
	return []string{"generic", "collectd", "ceilometer"}
}

//SetFromString resets value according to given human readable identification. Returns false if invalid identification was given.
func (src *DataSource) SetFromString(name string) bool {
	for index, value := range src.ListAll() {
		if name == value {
			*src = DataSource(index)
			return true
		}
	}
	return false
}

//String returns human readable data type identification.
func (src DataSource) String() string {
	return (src.ListAll())[src]
}

//Prefix returns human readable data type identification.
func (src DataSource) Prefix() string {
	return fmt.Sprintf("%s_*", src.String())
}
