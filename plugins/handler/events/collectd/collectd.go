package collectd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/infrawatch/sg-core/pkg/bus"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/infrawatch/sg-core/plugins/handler/events/pkg/lib"
	jsoniter "github.com/json-iterator/go"
)

//collectd contains objects for handling collectd events

var (
	// Regular expression for sanitizing received data
	rexForArray          = regexp.MustCompile(`^\[.*\]$`)
	rexForNestedQuote    = regexp.MustCompile(`\\\"`)
	rexForRemainedNested = regexp.MustCompile(`":"[^",]+\\\\\"[^",]+"`)
	rexForVes            = regexp.MustCompile(`"ves":"{(.*)}"`)
	rexForInvalidVesStr  = regexp.MustCompile(`":"[^",\\]+"[^",\\]+"`)

	json                  = jsoniter.ConfigCompatibleWithStandardLibrary
	collectdAlertSeverity = map[string]data.EventSeverity{
		"OKAY":    data.INFO,
		"WARNING": data.WARNING,
		"FAILURE": data.CRITICAL,
	}
)

const source string = "collectd"

type msgType int

const (
	collectd msgType = iota
	sensubility
)

type eventMessage struct {
	Labels      map[string]interface{}
	Annotations map[string]interface{}
	StartsAt    string `json:"startsAt"`
}

//Collectd type for handling collectd event messages
type Collectd struct {
	events []data.Event
}

//PublishEvents write events to publish func
func (c *Collectd) PublishEvents(epf bus.EventPublishFunc) {
	for _, e := range c.events {
		epf(e)
	}
}

//Parse parse event message
func (c *Collectd) Parse(blob []byte) error {
	message := []eventMessage{}
	err := json.UnmarshalFromString(sanitize(blob), &message)
	if err != nil {
		return err
	}

	// create index
	for _, eMsg := range message {
		var name string
		if value, ok := eMsg.Labels["alertname"].(string); ok {
			//gets rid of last term showing type like "gauge"
			if index := strings.LastIndex(value, "_"); index > len("collectd_") {
				name = value[0:index]
			} else {
				name = value
			}
		}

		var publisher string
		var ok bool
		publisher, ok = eMsg.Labels["instance"].(string)
		if !ok {
			publisher = "unknown"
		}
		if !strings.HasPrefix(name, fmt.Sprintf("%s_", "collectd")) {
			name = fmt.Sprintf("%s_%s", source, name)
		}

		var eSeverity data.EventSeverity
		if value, ok := eMsg.Labels["severity"]; ok {
			if severity, ok := collectdAlertSeverity[value.(string)]; ok {
				eSeverity = severity
			} else {
				eSeverity = data.UNKNOWN
			}
		} else {
			eSeverity = data.UNKNOWN
		}

		c.events = append(c.events, data.Event{
			Index:     name,
			Type:      data.EVENT,
			Severity:  eSeverity,
			Publisher: publisher,
			Time:      float64(lib.EpochFromFormat(eMsg.StartsAt)),
			Labels:    eMsg.Labels,
			Annotations: lib.AssimilateMap(eMsg.Annotations, map[string]interface{}{
				"source_type":  source,
				"processed_by": "sg",
			}),
		})
	}
	return nil
}

func sanitize(jsondata []byte) string {
	// sanitize "ves" field which can come in nested string in more than one level
	var output []byte
	if rexForArray.FindSubmatch(jsondata) == nil {
		// messages from collectd-sensubility don't contain array, so add surrounding brackets
		output = append([]byte("["), jsondata...)
		output = append(output, []byte("]")...)
	}
	sub := rexForVes.FindStringSubmatch(string(output))
	if len(sub) == 2 {
		substr := sub[1]
		for {
			cleaned := rexForNestedQuote.ReplaceAllString(substr, `"`)
			if rexForInvalidVesStr.FindString(cleaned) == "" {
				substr = cleaned
			}
			if rexForRemainedNested.FindString(cleaned) == "" {
				break
			}
		}
		res := rexForVes.ReplaceAllLiteralString(string(output), fmt.Sprintf(`"ves":{%s}`, substr))
		output = []byte(res)
	}
	return string(output)
}
