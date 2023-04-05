package log

import (
	"context"
)

type contextLogKeyType any

// Use a unique type so that there will never be a conflict with a different key
var contextLogKey contextLogKeyType

// LogContext will return a new context with the given logger added
// to the given context.
func LogContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextLogKey, logger.Clone())
}

// FromContext will extract a logger from a context if it contains one,
// or return the default logger if it doesn't.
func FromContext(ctx context.Context) Logger {
	if logger := ctx.Value(contextLogKey); logger != nil {
		return logger.(Logger)
	}
	return defaultLogger
}
