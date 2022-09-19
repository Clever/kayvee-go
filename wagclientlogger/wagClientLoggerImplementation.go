package wagclientlogger

import (
	"github.com/Clever/kayvee-go/v7/logger"
	wcl "github.com/Clever/wag/logging/wagclientlogger"
)

var kv logger.KayveeLogger

//NewLogger creates a logger for id that produces logs at and below the indicated level.
//Level indicated the level at and below which logs are created.
func NewLogger(id string, level wcl.LogLevel) WagClientLogger {
	newLogger := logger.New(id)

	if level != wcl.FromEnv {
		newLogger.SetLogLevel(logger.LogLevel(level))
	}

	kv = newLogger

	return WagClientLogger{id: id, level: level}
}

type WagClientLogger struct {
	level wcl.LogLevel
	id    string
}

func (w WagClientLogger) Log(level wcl.LogLevel, title string, m map[string]interface{}) {
	if title != "" {
		m["title"] = title
	}
	if kv == nil {
		NewLogger(title, wcl.FromEnv)
	}
	switch level {
	case wcl.Critical:
		kv.CriticalD(w.id, m)
	case wcl.Error:
		kv.ErrorD(w.id, m)
	case wcl.Warning:
		kv.WarnD(w.id, m)
	case wcl.Info:
		kv.InfoD(w.id, m)
	case wcl.Debug:
		kv.DebugD(w.id, m)
	case wcl.Trace:
		kv.TraceD(w.id, m)
	}
}
