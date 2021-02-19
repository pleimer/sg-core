package collectd

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/infrawatch/sg-core/pkg/data"
	jsoniter "github.com/json-iterator/go"
)

//collectd contains objects for handling collectd events

var (
	// Regular expression for sanitizing received data
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

	isoTimeLayout = "2006-01-02 15:04:05.000000"
	rFC3339       = "2006-01-02T15:04:05.000000"
)

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

//Parse parse event message
func Parse(blob []byte) (*data.Event, error) {
	message := eventMessage{}
	err := json.UnmarshalFromString(sanitize(blob), &message)
	if err != nil {
		fmt.Println(string(blob))
		return nil, err
	}

	// create index
	var name string
	if value, ok := message.Labels["alertname"].(string); ok {
		if index := strings.LastIndex(value, "_"); index > len("collectd_") {
			name = value[0:index]
		} else {
			name = value
		}
	}

	if !strings.HasPrefix(name, fmt.Sprintf("%s_", "collectd")) {
		name = fmt.Sprintf("%s_%s", "collectd", name)
	}

	var eSeverity data.EventSeverity
	if value, ok := message.Labels["severity"]; ok {
		if severity, ok := collectdAlertSeverity[value.(string)]; ok {
			eSeverity = severity
		} else {
			eSeverity = data.UNKNOWN
		}
	} else {
		eSeverity = data.UNKNOWN
	}

	return &data.Event{
		Index:       name,
		Type:        data.EVENT,
		Severity:    eSeverity,
		Time:        float64(epochFromFormat(message.StartsAt)),
		Labels:      message.Labels,
		Annotations: message.Annotations,
	}, nil
}

func sanitize(jsondata []byte) string {
	output := string(bytes.Trim(jsondata, "\t []"))
	// sanitize "ves" field which can come in nested string in more than one level
	sub := rexForVes.FindStringSubmatch(output)
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
		output = rexForVes.ReplaceAllLiteralString(output, fmt.Sprintf(`"ves":{%s}`, substr))
	}
	return output
}

func epochFromFormat(ts string) int64 {
	for _, layout := range []string{rFC3339, time.RFC3339, time.RFC3339Nano, time.ANSIC, isoTimeLayout} {
		stamp, err := time.Parse(layout, ts)
		if err == nil {
			return stamp.Unix()
		}
	}
	return 0
}
