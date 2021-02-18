package lib

import (
	"fmt"
	"regexp"
)

var (
	// Regular expression for sanitizing received data
	rexForNestedQuote    = regexp.MustCompile(`\\\"`)
	rexForRemainedNested = regexp.MustCompile(`":"[^",]+\\\\\"[^",]+"`)
	rexForVes            = regexp.MustCompile(`"ves":"{(.*)}"`)
	rexForInvalidVesStr  = regexp.MustCompile(`":"[^",\\]+"[^",\\]+"`)
	rexForCleanPayload   = regexp.MustCompile(`\"payload\"\s*:\s*\[(.*)\]`)
)

func sanitizeGeneric(jsondata []byte) string {
	return string(jsondata)
}

func sanitizeCeilometer(jsondata []byte) string {
	sanitized := string(jsondata)
	// parse only relevant data
	sub := rexForOsloMessage.FindStringSubmatch(sanitized)
	sanitized = rexForNestedQuote.ReplaceAllString(sub[1], `"`)
	// avoid getting payload data wrapped in array
	item := rexForCleanPayload.FindStringSubmatch(sanitized)
	if len(item) == 2 {
		sanitized = rexForCleanPayload.ReplaceAllLiteralString(sanitized, fmt.Sprintf(`"payload":%s`, item[1]))
	}
	return sanitized
}

func sanitizeCollectd(jsondata []byte) string {
	output := string(jsondata)
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

//Sanitizers for each data source
var Sanitizers = map[string](func([]byte) string){
	"ceilometer": sanitizeCeilometer,
	"collectd":   sanitizeCollectd,
	"generic":    sanitizeGeneric,
}
