package kayvee

import (
	"encoding/json"
	"fmt"
)

// Log Levels:

type LogLevel string

const (
	Unknown  LogLevel = "unknown"
	Critical          = "critical"
	Error             = "error"
	Warning           = "warning"
	Info              = "info"
	Trace             = "trace"
)

// Format converts a map to a string of space-delimited key=val pairs
func Format(data map[string]interface{}) string {
	formattedString, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Error converting kayvee message to json %s", err.Error())
	}
	return string(formattedString)
}

// FormatLog is similar to Format, but takes additional reserved params to promote logging best-practices
func FormatLog(source string, level LogLevel, title string, data map[string]interface{}) string {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["source"] = source
	data["level"] = level
	data["title"] = title
	return Format(data)
}
