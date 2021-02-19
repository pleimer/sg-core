package lib

import "fmt"

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
