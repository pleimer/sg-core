package lib

import (
	"fmt"
	"strings"
	"time"
)

const (
	alertSource     = "SmartGateway"
	isoTimeLayout   = "2006-01-02 15:04:05.000000"
	unknownSeverity = "unknown"
)

var (
	collectdAlertSeverity = map[string]string{
		"OKAY":    "info",
		"WARNING": "warning",
		"FAILURE": "critical",
	}
	ceilometerAlertSeverity = map[string]string{
		"audit":    "info",
		"info":     "info",
		"sample":   "info",
		"warn":     "warning",
		"warning":  "warning",
		"critical": "critical",
		"error":    "critical",
		"AUDIT":    "info",
		"INFO":     "info",
		"SAMPLE":   "info",
		"WARN":     "warning",
		"WARNING":  "warning",
		"CRITICAL": "critical",
		"ERROR":    "critical",
	}
)

//alertKeySurrogate translates case for fields for AlertManager
type alertKeySurrogate struct {
	Parsed string
	Label  string
}

func formatTimestamp(ts string) string {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, time.ANSIC, isoTimeLayout} {
		stamp, err := time.Parse(layout, ts)
		if err == nil {
			return stamp.Format(time.RFC3339)
		}
	}
	return ""
}

//GenerateCollectdAlert generates PrometheusAlert from the event data
func generateCollectdAlert(generatorURL string, evt map[string]interface{}) *PrometheusAlert {
	alert := &PrometheusAlert{
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
		GeneratorURL: generatorURL,
	}
	assimilateMap(evt["labels"].(map[string]interface{}), &alert.Labels)
	assimilateMap(evt["annotations"].(map[string]interface{}), &alert.Annotations)
	if value, ok := evt["startsAt"].(string); ok {
		// ensure timestamps is in RFC3339
		alert.StartsAt = formatTimestamp(value)
	}

	if value, ok := alert.Labels["severity"]; ok {
		if severity, ok := collectdAlertSeverity[value]; ok {
			alert.Labels["severity"] = severity
		} else {
			alert.Labels["severity"] = unknownSeverity
		}
	} else {
		alert.Labels["severity"] = unknownSeverity
	}

	// generate SG-relevant data
	alert.SetName()
	alert.SetSummary()
	alert.Labels["alertsource"] = alertSource
	return alert
}

func generateCeilometerAlert(generatorURL string, evt map[string]interface{}) *PrometheusAlert {
	alert := &PrometheusAlert{
		Labels:       make(map[string]string),
		Annotations:  make(map[string]string),
		GeneratorURL: generatorURL,
	}
	// set labels
	alertName := "ceilometer_generic"
	if payload, ok := evt["payload"]; ok {
		if typedPayload, ok := payload.(map[string]interface{}); ok {
			if val, ok := typedPayload["event_type"]; ok {
				if strVal, ok := val.(string); ok {
					alertName = strVal
				}
			}
		}
	}
	if alertName == "ceilometer_generic" {
		if val, ok := evt["event_type"]; ok {
			if strVal, ok := val.(string); ok {
				alertName = strVal
			}
		}
	}
	// replace dotted notation and dashes with underscores, omit last item (gauge, ...)
	parts := strings.Split(alertName, ".")
	if len(parts) > 1 {
		alertName = strings.Join(parts[:len(parts)-1], "_")
	}
	alertName = strings.ReplaceAll(alertName, "-", "_")
	// ensure name is prefixed with source name
	if !strings.HasPrefix(alertName, "ceilometer_") {
		alertName = fmt.Sprintf("ceilometer_%s", alertName)
	}
	alert.Labels["alertname"] = alertName

	surrogates := []alertKeySurrogate{
		{"message_id", "messageId"},
		{"publisher_id", "instance"},
		{"event_type", "type"},
	}
	for _, renameCase := range surrogates {
		if value, ok := evt[renameCase.Parsed]; ok {
			alert.Labels[renameCase.Label] = value.(string)
		}
	}
	if value, ok := evt["priority"]; ok {
		if severity, ok := ceilometerAlertSeverity[value.(string)]; ok {
			alert.Labels["severity"] = severity
		} else {
			alert.Labels["severity"] = unknownSeverity
		}
	} else {
		alert.Labels["severity"] = unknownSeverity
	}
	if value, ok := evt["publisher_id"].(string); ok {
		alert.Labels["sourceName"] = fmt.Sprintf("ceilometer@%s", value)
	}
	assimilateMap(evt["payload"].(map[string]interface{}), &alert.Annotations)
	// set timestamp
	if value, ok := evt["timestamp"].(string); ok {
		// ensure timestamp is in RFC3339
		alert.StartsAt = formatTimestamp(value)
	}
	// generate SG-relevant data
	alert.SetName()
	alert.SetSummary()
	alert.Labels["alertsource"] = alertSource
	return alert
}

//AlertGenerators is a map of PrometheusAlert generators
var AlertGenerators = map[string](func(string, map[string]interface{}) *PrometheusAlert){
	"collectd":   generateCollectdAlert,
	"ceilometer": generateCeilometerAlert,
}
