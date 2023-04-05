package log

import (
	"github.com/aws/smithy-go/logging"
)

type awsLogger struct {
	ddl DynamicDefaultLogger
}

func (l *awsLogger) Logf(classification logging.Classification, template string, args ...interface{}) {
	switch classification {
	case logging.Debug:
		l.ddl.Logger().Debugf(template, args...)
	case logging.Warn:
		l.ddl.Logger().Warnf(template, args...)
	default:
		l.ddl.Logger().Infof(template, args...)
	}
}

func GetAwsLogger() logging.Logger {
	return &awsLogger{
		ddl: NewDynamicDefaultLogger(func(in NewInput) NewInput {
			// Add one skipped frame for this logger
			in.SkippedFrames += 1
			return in
		}),
	}
}
