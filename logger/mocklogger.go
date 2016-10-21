package logger

import (
	"io"
)

// MockRouteCountLogger is a mock implementation of KayveeLogger that counts the router rules
// applied to each log call without actually formatting or writing the log line.
type MockRouteCountLogger struct {
	logger      *Logger
	routeCounts map[string]int
}

// GetRuleCounts returns a map of rule names to the number of times that rule has been applied
// in routing logs for MockRouteCountLogger. Only includes routing rules that have at least
// one use.
func (ml *MockRouteCountLogger) GetRuleCounts() map[string]int {
	out := make(map[string]int)
	for k, v := range ml.routeCounts {
		out[k] = v
	}
	return out
}

// NewMockCountLogger returns a new MockRoutCountLogger with the specified `source`.
func NewMockCountLogger(source string) *MockRouteCountLogger {
	return NewMockCountLoggerWithContext(source, nil)
}

// NewMockCountLoggerWithContext returns a new MockRoutCountLogger with the specified `source` and `contextValues`.
func NewMockCountLoggerWithContext(source string, contextValues map[string]interface{}) *MockRouteCountLogger {
	routeCounts := make(map[string]int)
	lg := NewWithContext(source, contextValues)
	lg.fLogger = &routeCountingFormatLogger{
		routeCounts: routeCounts,
	}
	mocklg := MockRouteCountLogger{
		logger:      lg,
		routeCounts: routeCounts,
	}
	return &mocklg
}

/////////////////////////////
//
// routeCountingFormatLogger
//
/////////////////////////////

// routeCountingFormatLogger implements the formatLogger interface to allow for counting
// invocations of routing rules.
type routeCountingFormatLogger struct {
	routeCounts map[string]int
}

// formatAndLog tracks routing statistics for this mock router.
// Initialization works as with the default format logger, but no formatting or logging is actually performed.
func (fl *routeCountingFormatLogger) formatAndLog(data map[string]interface{}) {
	routeData, ok := data["_kvmeta"]
	if !ok {
		return
	}
	routes, ok := routeData.(map[string]interface{})["routes"]
	if !ok {
		return
	}
	for _, route := range routes.([]map[string]interface{}) {
		rule := route["rule"].(string)
		fl.routeCounts[rule] = fl.routeCounts[rule] + 1
	}
}

// setFormatter implements the FormatLogger method.
func (fl *routeCountingFormatLogger) setFormatter(formatter Formatter) {
	// we don't format anything in this mock logger
	return
}

// setOutput implements the FormatLogger method.
func (fl *routeCountingFormatLogger) setOutput(output io.Writer) {
	// we don't output anything in this mock logger
	return
}

/////////////////////////////////////////////////////////////
//
//	KayveeLogger implementation (all passthrough to logger)
//
/////////////////////////////////////////////////////////////

// SetConfig implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) SetConfig(source string, logLvl LogLevel, formatter Formatter, output io.Writer) {
	ml.logger.SetConfig(source, logLvl, formatter, output)
}

// AddContext implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) AddContext(key, val string) {
	ml.logger.AddContext(key, val)
}

// SetLogLevel implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) SetLogLevel(logLvl LogLevel) {
	ml.logger.SetLogLevel(logLvl)
}

// SetFormatter implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) SetFormatter(formatter Formatter) {
	ml.logger.SetFormatter(formatter)
}

// SetOutput implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) SetOutput(output io.Writer) {
	ml.logger.SetOutput(output)
}

// Debug implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) Debug(title string) {
	ml.logger.Debug(title)
}

// Info implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) Info(title string) {
	ml.logger.Info(title)
}

// Warn implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) Warn(title string) {
	ml.logger.Warn(title)
}

// Error implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) Error(title string) {
	ml.logger.Error(title)
}

// Critical implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) Critical(title string) {
	ml.logger.Critical(title)
}

// Counter implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) Counter(title string) {
	ml.logger.Counter(title)
}

// GaugeInt implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) GaugeInt(title string, value int) {
	ml.logger.GaugeInt(title, value)
}

// GaugeFloat implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) GaugeFloat(title string, value float64) {
	ml.logger.GaugeFloat(title, value)
}

// DebugD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) DebugD(title string, data map[string]interface{}) {
	ml.logger.DebugD(title, data)
}

// InfoD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) InfoD(title string, data map[string]interface{}) {
	ml.logger.InfoD(title, data)
}

// WarnD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) WarnD(title string, data map[string]interface{}) {
	ml.logger.WarnD(title, data)
}

// ErrorD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) ErrorD(title string, data map[string]interface{}) {
	ml.logger.ErrorD(title, data)
}

// CriticalD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) CriticalD(title string, data map[string]interface{}) {
	ml.logger.CriticalD(title, data)
}

// CounterD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) CounterD(title string, value int, data map[string]interface{}) {
	ml.logger.CounterD(title, value, data)
}

// GaugeIntD implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) GaugeIntD(title string, value int, data map[string]interface{}) {
	ml.logger.GaugeIntD(title, value, data)
}

// GaugeFloatD implements the method for the KayveeLogger interface.
// Logs with type = gauge, and value = value
func (ml *MockRouteCountLogger) GaugeFloatD(title string, value float64, data map[string]interface{}) {
	ml.logger.GaugeFloatD(title, value, data)
}

// WithRoutingConfig implements the method for the KayveeLogger interface.
func (ml *MockRouteCountLogger) WithRoutingConfig(filename string) (KayveeLogger, error) {
	return ml.logger.WithRoutingConfig(filename)
}
