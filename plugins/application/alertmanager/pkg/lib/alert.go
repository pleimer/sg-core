package lib

import (
	"sort"
	"strings"
)

//PrometheusAlert represents data structure used for sending alerts to Prometheus Alert Manager
type PrometheusAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt,omitempty"`
	EndsAt       string            `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL"`
}

//SetName generates unique name and description for the alert and creates new key/value pair for it in Labels
func (alert *PrometheusAlert) SetName() {
	if _, ok := alert.Labels["name"]; !ok {
		keys := make([]string, 0, len(alert.Labels))
		for k := range alert.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		values := make([]string, 0, len(alert.Labels)-1)
		desc := make([]string, 0, len(alert.Labels))
		for _, k := range keys {
			if k != "severity" {
				values = append(values, alert.Labels[k])
			}
			desc = append(desc, alert.Labels[k])
		}
		alert.Labels["name"] = strings.Join(values, "_")
		alert.Annotations["description"] = strings.Join(desc, " ")
	}
}

//SetSummary generates summary annotation in case it is empty
func (alert *PrometheusAlert) SetSummary() {
	generate := false
	if _, ok := alert.Annotations["summary"]; ok {
		if alert.Annotations["summary"] == "" {
			generate = true
		}
	} else {
		generate = true
	}

	if generate {
		if val, ok := alert.Labels["summary"]; ok && alert.Labels["summary"] != "" {
			alert.Annotations["summary"] = val
		} else {
			values := make([]string, 0, 3)
			for _, key := range []string{"sourceName", "type", "eventName"} {
				if val, ok := alert.Labels[key]; ok {
					values = append(values, val)
				}
			}
			alert.Annotations["summary"] = strings.Join(values, " ")
		}
	}
}