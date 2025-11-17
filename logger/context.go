package logger

import (
	"context"
)

type loggerKeyType struct{}

var loggerKey = loggerKeyType{}

type contextNameKey string

// A special context key for which the value will be automatically injected into
// the logger context when a logger is accessed with FromContext.
const ContextName contextNameKey = "logger.contextname"

// NewContext creates a new context object containing a logger value.
func NewContext(ctx context.Context, logger KayveeLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger value contained in a context.
// For convenience, if the context does not contain a logger, a new logger is
// created and returned. This allows users of this method to use the logger
// immediately, e.g.
//
//	logger.FromContext(ctx).Info("...")
//
// When used with a context that has ContextNameKey set, will automatically
// include this value in the logger context as context_name.
func FromContext(ctx context.Context) KayveeLogger {
	value := ctx.Value(loggerKey)

	var logger *Logger
	var ok bool
	if logger, ok = value.(*Logger); !ok {
		logger = NewConcreteLogger("")
	}

	if name, ok := ctx.Value(ContextName).(string); ok && name != "" {
		logger.globalsL.Lock()
		defer logger.globalsL.Unlock()
		logger.globals["context_name"] = name
	}

	return logger
}
