package kayvee

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var (
	deployEnv    string
	workflowID   string
	podID        string
	podShortname string
	podRegion    string
	podAccount   string
)

func init() {
	if v := os.Getenv("_DEPLOY_ENV"); v != "" {
		deployEnv = v
	} else if v := os.Getenv("DEPLOY_ENV"); v != "" {
		deployEnv = v
	}
	if os.Getenv("_EXECUTION_NAME") != "" {
		workflowID = os.Getenv("_EXECUTION_NAME")
	}
	if os.Getenv("_POD_ID") != "" {
		podID = os.Getenv("_POD_ID")
	}
	if os.Getenv("_POD_SHORTNAME") != "" {
		podShortname = os.Getenv("_POD_SHORTNAME")
	}
	if os.Getenv("_POD_REGION") != "" {
		podRegion = os.Getenv("_POD_REGION")
	}
	if os.Getenv("_POD_ACCOUNT") != "" {
		podAccount = os.Getenv("_POD_ACCOUNT")
	}
}

// Log Levels:

// LogLevel denotes the level of a logging
type LogLevel string

// Constants used to define different LogLevels supported
const (
	Unknown  LogLevel = "unknown"
	Critical          = "critical"
	Error             = "error"
	Warning           = "warning"
	Info              = "info"
	Trace             = "trace"
)

// Internal defaults used by Logger.
const (
	defaultFlags = log.LstdFlags | log.Lshortfile
)

// Format converts a map to a string of space-delimited key=val pairs
func Format(data map[string]interface{}) string {
	if deployEnv != "" {
		data["deploy_env"] = deployEnv
	}
	if workflowID != "" {
		data["wf_id"] = workflowID
	}
	if podID != "" {
		data["pod-id"] = podID
	}
	if podShortname != "" {
		data["pod-shortname"] = podShortname
	}
	if podRegion != "" {
		data["pod-region"] = podRegion
	}
	if podAccount != "" {
		data["pod-account"] = podAccount
	}
	formattedString, err := json.Marshal(data)
	if err != nil {
		for k, v := range data {
			_, err := json.Marshal(v)
			if err != nil {
				data[k] = fmt.Sprintf("Error marshaling value in map, err: %s, value: %+v", err.Error(), v)
			}
		}
		formattedString, _ = json.Marshal(data)
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

// Logger is an interface satisfied by all loggers that use kayvee to Log results
type Logger interface {
	Info(title string, data map[string]interface{})
	Warning(title string, data map[string]interface{})
	Error(title string, data map[string]interface{}, err error)
}
