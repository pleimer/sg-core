package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"
	"github.com/pkg/errors"

	"github.com/infrawatch/sg-core/plugins/application/elasticsearch/pkg/lib"
)

const (
	appname       = "elasticsearch"
	genericSuffix = "_generic"
)

//wrapper object for elasitcsearch index
type esIndex struct {
	index  string
	record []string
}

//Elasticsearch plugin saves events to Elasticsearch database
type Elasticsearch struct {
	configuration *lib.AppConfig
	logger        *logging.Logger
	client        *lib.Client
	buffer        map[string][]string
	dump          chan esIndex
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &Elasticsearch{
		logger: logger,
		buffer: make(map[string][]string),
		dump:   make(chan esIndex, 100),
	}
}

//getIndexName returns ES index name appropriate to data source and event type
func getIndexName(record map[string]interface{}, source string) string {
	output := fmt.Sprintf("%s%s", source, genericSuffix)

	switch source {
	case "collectd":
		if val, ok := record["labels"]; ok {
			if labels, ok := val.(map[string]interface{}); ok {
				if value, ok := labels["alertname"].(string); ok {
					if index := strings.LastIndex(value, "_"); index > len("collectd_") {
						output = value[0:index]
					} else {
						output = value
					}
				}
			}
		}
	case "ceilometer":
		// use event_type from payload or fallback to message's event_type if N/A
		if payload, ok := record["payload"]; ok {
			if typedPayload, ok := payload.(map[string]interface{}); ok {
				if val, ok := typedPayload["event_type"]; ok {
					if strVal, ok := val.(string); ok {
						output = strVal
					}
				}
			}
		}
		if output == fmt.Sprintf("%s%s", source, genericSuffix) {
			if val, ok := record["event_type"]; ok {
				if strVal, ok := val.(string); ok {
					output = strVal
				}
			}
		}
		// replace dotted notation and dashes with underscores
		parts := strings.Split(output, ".")
		if len(parts) > 1 {
			output = strings.Join(parts[:len(parts)-1], "_")
		}
		output = strings.ReplaceAll(output, "-", "_")
	}

	// ensure index name is prefixed with source name
	if !strings.HasPrefix(output, fmt.Sprintf("%s_", source)) {
		output = fmt.Sprintf("%s_%s", source, output)
	}
	return output
}

//ReceiveEvent receive event from event bus
func (es *Elasticsearch) ReceiveEvent(hName string, eType data.EventType, event map[string]interface{}) {
	switch eType {
	case data.ERROR:
		//TODO: error handling
	case data.EVENT:
		// get data source
		src, ok := event["source"]
		if !ok {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
			es.logger.Warn("internal event does not contain source information - disregarding")
			return
		}
		source, ok := src.(string)
		if !ok {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
			es.logger.Warn("invalid format of source information - disregarding")
			return
		}
		// get record
		message, ok := event["message"]
		if !ok {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "event": event})
			es.logger.Warn("internal event does not contain record data - disregarding")
			return
		}
		rec, ok := message.(map[string]interface{})
		if !ok {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "record": rec})
			es.logger.Warn("received incorrectly formatted record - disregarding")
			return
		}
		// mashal record for storage
		record, err := json.Marshal(rec)
		if err != nil {
			es.logger.Metadata(logging.Metadata{"plugin": appname, "record": rec, "error": err})
			es.logger.Error("failed to marshal record - disregarding")
			return
		}
		// buffer or index record
		index := getIndexName(rec, source)
		var recordList []string
		if es.configuration.BufferSize > 1 {
			if _, ok := es.buffer[index]; !ok {
				es.buffer[index] = make([]string, 0, es.configuration.BufferSize)
			}
			es.buffer[index] = append(es.buffer[index], string(record))
			if len(es.buffer[index]) < es.configuration.BufferSize {
				// buffer is not full, don't send
				es.logger.Metadata(logging.Metadata{"plugin": appname, "record": rec})
				es.logger.Debug("Buffering record")
				return
			}
			recordList = es.buffer[index]
			delete(es.buffer, index)
		} else {
			recordList = []string{string(record)}
		}
		es.dump <- esIndex{index: index, record: recordList}
	case data.RESULT:
		//TODO: sensubility result handling
	case data.LOG:
		//TODO: log collection handling
	}

}

//Run plugin process
func (es *Elasticsearch) Run(ctx context.Context, done chan bool) {
	if es.configuration.ResetIndex {
		es.client.IndicesDelete([]string{"generic_*", "collectd_*", "ceilometer_*"})
	}
	es.logger.Metadata(logging.Metadata{"plugin": appname, "url": es.configuration.HostURL})
	es.logger.Info("storing events to Elasticsearch.")

	for {
		select {
		case <-ctx.Done():
			goto done
		case dumped := <-es.dump:
			if err := es.client.Index(dumped.index, dumped.record, es.configuration.BulkIndex); err != nil {
				es.logger.Metadata(logging.Metadata{"plugin": appname, "event": dumped.record, "error": err})
				es.logger.Error("failed to index event - disregarding")
			} else {
				es.logger.Debug("successfully indexed document(s)")
			}
		}
	}

done:
	es.logger.Metadata(logging.Metadata{"plugin": appname})
	es.logger.Info("exited")
}

//Config implements application.Application
func (es *Elasticsearch) Config(c []byte) error {
	es.configuration = &lib.AppConfig{
		HostURL:       "",
		UseTLS:        false,
		TLSServerName: "",
		TLSClientCert: "",
		TLSClientKey:  "",
		TLSCaCert:     "",
		UseBasicAuth:  false,
		User:          "",
		Password:      "",
		ResetIndex:    false,
		BufferSize:    1,
		BulkIndex:     false,
	}
	err := config.ParseConfig(bytes.NewReader(c), es.configuration)
	if err != nil {
		return err
	}

	es.client, err = lib.NewElasticClient(es.configuration)
	if err != nil {
		return errors.Wrap(err, "failed to connect to Elasticsearch host")
	}
	return nil
}
