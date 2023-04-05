package retryablehttp

import (
	"context"
	"errors"

	"github.com/Invicton-Labs/go-common/log"
	"github.com/hashicorp/go-retryablehttp"
)

type retryhttpLeveledLogger struct {
	ddl log.DynamicDefaultLogger
}

func (l *retryhttpLeveledLogger) Error(msg string, keysAndValues ...interface{}) {
	hasErrorKey := false
	logger := l.ddl.Logger()
	// Check if any of the fields are an error
	for i := 0; i < len(keysAndValues); i += 2 {
		if keysAndValues[i] == "error" && len(keysAndValues) > i+1 {
			hasErrorKey = true
			// If it's a context error, log it at debug level instead of the default error level.
			// Ehen contexts are cancelled, we generally only care about the reason they're cancelled,
			// not the effects of the cancellation.
			if err, ok := keysAndValues[i+1].(error); ok && errors.Is(err, context.Canceled) {
				logger.Debugw(msg, keysAndValues...)
				return
			}
		}
	}
	// If it's a connection error, which will be retried, we really don't care too much.
	// Just log it at the debug level.
	if hasErrorKey {
		for i := 0; i < len(keysAndValues); i += 2 {
			if keysAndValues[i] == "url" {
				logger.Debugw(msg, keysAndValues...)
				return
			}
		}
	}
	logger.Errorw(msg, keysAndValues...)
}
func (l *retryhttpLeveledLogger) Info(msg string, keysAndValues ...interface{}) {
	l.ddl.Logger().Infow(msg, keysAndValues...)
}
func (l *retryhttpLeveledLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.ddl.Logger().Debugw(msg, keysAndValues...)
}
func (l *retryhttpLeveledLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.ddl.Logger().Warnw(msg, keysAndValues...)
}

func GetRetryhttpLeveledLogger(loggerConfigFunc func(log.NewInput) log.NewInput) retryablehttp.LeveledLogger {
	return &retryhttpLeveledLogger{
		ddl: log.NewDynamicDefaultLogger(func(input log.NewInput) log.NewInput {
			input.InitialFields["retryable_http"] = true
			input.SkippedFrames += 1
			if loggerConfigFunc != nil {
				input = loggerConfigFunc(input)
			}
			return input
		}),
	}
}
