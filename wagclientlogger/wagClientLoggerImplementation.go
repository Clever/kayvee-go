package wagclientlogger

import (
	"github.com/Clever/kayvee-go/v7/logger"
)

var kv logger.KayveeLogger

//NewLogger creates a logger for id that produces logs at and below the indicated level.
//Level indicated the level at and below which logs are created.
func NewLogger(id string, level LogLevel) WagClientLogger {
	newLogger := logger.New(id)

	if level != FromEnv {
		newLogger.SetLogLevel(logger.LogLevel(level))
	}

	kv = newLogger

	return WagClientLogger{id: id, level: level}
}

type WagClientLogger struct {
	level LogLevel
	id    string
}

func (w WagClientLogger) Log(level LogLevel, message string, m map[string]interface{}) {
	m["message"] = message
	switch level {
	case Critical:
		kv.CriticalD(w.id, m)
	case Error:
		kv.ErrorD(w.id, m)
	case Warning:
		kv.WarnD(w.id, m)
	case Info:
		kv.InfoD(w.id, m)
	case Debug:
		kv.DebugD(w.id, m)
	case Trace:
		kv.TraceD(w.id, m)
	}
}

type LogLevel int

// Constants used to define different LogLevels supported
const (
	Trace LogLevel = iota
	Debug
	Info
	Warning
	Error
	Critical
	FromEnv
)

var logLevelNames = map[LogLevel]string{
	Trace:    "trace",
	Debug:    "debug",
	Info:     "info",
	Warning:  "warning",
	Error:    "error",
	Critical: "critical",
	FromEnv:  "from environment vars",
}

func (l LogLevel) String() string {
	if s, ok := logLevelNames[l]; ok {
		return s
	}
	return ""
}

func strLvlToInt(s string) int {
	switch s {
	case "Critical":
		return 5
	case "Error":
		return 4
	case "Warning":
		return 3
	case "Info":
		return 2
	case "Debug":
		return 1
	case "Trace":
		return 0
	}
	return -1
}
