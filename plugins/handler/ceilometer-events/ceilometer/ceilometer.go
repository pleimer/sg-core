package ceilometer

import (
	"fmt"
	"regexp"

	jsoniter "github.com/json-iterator/go"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

var (
	// Regular expression for sanitizing received data
	rexForNestedQuote = regexp.MustCompile(`\\\"`)
)

type rawMessage struct {
	Request struct {
		OsloVersion string `json:"oslo.version"`
		OsloMessage string `json:"oslo.message"`
	}
}

func (rm *rawMessage) sanitizeMessage() {
	// sets oslomessage to cleaned state
	rm.Request.OsloMessage = rexForNestedQuote.ReplaceAllLiteralString(rm.Request.OsloMessage, `"`)
}

type osloMessage struct {
	Payload []struct {
		MessageID string `json:"message_id"`
		EventType string `json:"event_type"`
		Traits    []interface{}
	}
}

func (om *osloMessage) fromBytes(blob []byte) error {
	return json.Unmarshal(blob, om)
}

//Parse parse ceilometer message data
func Parse(blob []byte) error {
	rm := rawMessage{}
	json.Unmarshal(blob, &rm)
	rm.sanitizeMessage()

	om := osloMessage{}
	err := json.Unmarshal([]byte(rm.Request.OsloMessage), &om)
	if err != nil {
		return err
	}

	for _, payload := range om.Payload {
		ts, err := ceilometerTraitsFormatter(payload.Traits)
		if err != nil {
			return err
		}
		fmt.Printf("%v\n", ts)
	}

	return err
}

func ceilometerTraitsFormatter(traits []interface{}) (map[string]interface{}, error) {
	// transforms traits key into map[string]interface{}
	newTraits := make(map[string]interface{})
	for _, value := range traits {
		if typedValue, ok := value.([]interface{}); ok {
			if len(typedValue) != 3 {
				return nil, fmt.Errorf("parsed invalid trait in event: '%v'", value)
			}
			if traitType, ok := typedValue[1].(float64); ok {
				switch traitType {
				case 2:
					newTraits[typedValue[0].(string)] = typedValue[2].(float64)
				default:
					newTraits[typedValue[0].(string)] = typedValue[2].(string)
				}
			} else {
				return nil, fmt.Errorf("parsed invalid trait in event: '%v'", value)
			}
		} else {
			return nil, fmt.Errorf("parsed invalid trait in event: '%v'", value)
		}
	}
	return newTraits, nil
}
