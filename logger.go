package kayvee

import (
	logrus "github.com/nathanleiby/logrus"
	"os"
)

// F (Fields) in an abbreviation for map[string]interface, when logging using WithFields
type F map[string]interface{}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.DebugLevel)
}

// Info outputs a log with level "info"
func Info(msg string) {
	logrus.Info(msg)
}

// Warning outputs a log with level "warning"
func Warning(msg string) {
	logrus.Warning(msg)
}

// Error outputs a log with level "error"
func Error(msg string) {
	logrus.Error(msg)
}

// WithFields allows adding structured data to a log by passing a single key and value
// Note that it doesn't log until you call Info, Warning, or Error on its return value
func WithField(key string, value interface{}) *logrus.Entry {
	return logrus.WithField(key, value)
}

// WithFields allows adding structured data to a log by passing a map
// Note that it doesn't log until you call Info, Warning, or Error on its return value
func WithFields(f map[string]interface{}) *logrus.Entry {
	return logrus.WithFields(f)
}

// SetContext adds the set fields to every log message
func SetContext(c map[string]interface{}) {
	logrus.SetContext(c)
}

// GetContext returns the current context
func GetContext() logrus.Fields {
	return logrus.GetContext()
}
