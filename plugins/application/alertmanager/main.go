package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/infrawatch/apputils/logging"
	"github.com/infrawatch/sg-core/pkg/application"
	"github.com/infrawatch/sg-core/pkg/config"
	"github.com/infrawatch/sg-core/pkg/data"

	"github.com/infrawatch/sg-core/plugins/application/alertmanager/pkg/lib"
)

const (
	appname        = "alertmanager"
	handlersSuffix = "-events"
)

//AlertManager plugin suites for reporting alerts for Prometheus' alert manager
type AlertManager struct {
	configuration lib.AppConfig
	logger        *logging.Logger
	dump          chan lib.PrometheusAlert
}

//New constructor
func New(logger *logging.Logger) application.Application {
	return &AlertManager{
		configuration: lib.AppConfig{
			AlertManagerURL: "http://localhost",
			GeneratorURL:    "http://sg.localhost.localdomain",
		},
		logger: logger,
		dump:   make(chan lib.PrometheusAlert, 100),
	}
}

//ReceiveEvent is called whenever an event is broadcast on the event bus. The order of arguments
func (am *AlertManager) ReceiveEvent(hName string, eType data.EventType, msg string) {
	switch eType {
	case data.ERROR:
		//TODO: error handling
	case data.EVENT:
		// event handling
		if strings.HasSuffix(hName, handlersSuffix) {
			source := data.DataSource(0)
			if ok := source.SetFromString(hName[0:(len(hName) - len(handlersSuffix))]); !ok {
				am.logger.Metadata(logging.Metadata{"plugin": appname, "source": hName})
				am.logger.Warn("received event from unknown data source - disregarding")
			} else {
				record := make(map[string]interface{})
				err := json.Unmarshal([]byte(msg), &record)
				if err != nil {
					am.logger.Metadata(logging.Metadata{"plugin": appname, "event": msg, "error": err})
					am.logger.Error("failed to unmarshal event - disregarding")
				} else {
					// format message if needed
					err := data.EventFormatters[source.String()](record)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "event": record, "error": err})
						am.logger.Error("failed to format event - disregarding")
					} else {
						if generator, ok := lib.AlertGenerators[source.String()]; ok {
							am.dump <- *(generator(am.configuration.GeneratorURL, record))
						} else {
							am.logger.Metadata(logging.Metadata{"plugin": appname, "source": source.String()})
							am.logger.Error("missing alert generator for data source - disregarding")
						}
					}
				}
			}
		} else {
			am.logger.Metadata(logging.Metadata{"plugin": appname, "event": msg})
			am.logger.Info("received unknown data in event bus - disregarding")
		}
	case data.RESULT:
		//TODO: sensubility result handling
	case data.LOG:
		//TODO: log collection handling
	}

}

//Run implements main process of the application
func (am *AlertManager) Run(ctx context.Context, done chan bool) {
	wg := sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			goto done
		case dumped := <-am.dump:
			wg.Add(1)
			go func(url string, dumped lib.PrometheusAlert, logger *logging.Logger, wg *sync.WaitGroup) {
				defer wg.Done()
				alert, err := json.Marshal(dumped)
				if err != nil {
					logger.Metadata(logging.Metadata{"plugin": appname, "alert": dumped})
					logger.Warn("failed to marshal alert - disregarding")
				} else {
					buff := bytes.NewBufferString("[")
					buff.Write(alert)
					buff.WriteString("]")

					req, _ := http.NewRequest("POST", url, buff)
					req.Header.Set("X-Custom-Header", "smartgateway")
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						am.logger.Metadata(logging.Metadata{"plugin": appname, "error": err, "alert": buff.String()})
						am.logger.Error("failed to report alert to AlertManager")
					} else {
						// https://github.com/prometheus/alertmanager/blob/master/api/v2/openapi.yaml#L170
						if resp.StatusCode != http.StatusOK {
							body, _ := ioutil.ReadAll(resp.Body)
							resp.Body.Close()
							am.logger.Metadata(logging.Metadata{
								"plugin": appname,
								"status": resp.Status,
								"header": resp.Header,
								"body":   string(body)})
							am.logger.Error("failed to report alert to AlertManager")
						}
					}
				}
			}(am.configuration.AlertManagerURL, dumped, am.logger, &wg)
		}
	}

done:
	wg.Wait()
	am.logger.Metadata(logging.Metadata{"plugin": appname})
	am.logger.Info("exited")
}

//Config implements application.Application
func (am *AlertManager) Config(c []byte) error {
	am.configuration = lib.AppConfig{
		AlertManagerURL: "http://localhost",
		GeneratorURL:    "http://sg.localhost.localdomain",
	}
	err := config.ParseConfig(bytes.NewReader(c), &am.configuration)
	if err != nil {
		return err
	}
	return nil
}
