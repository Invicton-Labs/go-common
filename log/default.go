package log

import (
	"context"
	"sync"

	iselambda "github.com/Invicton-Labs/go-common/aws/lambda"
	"github.com/Invicton-Labs/go-common/collections"
	"github.com/Invicton-Labs/go-common/gensync"
	"github.com/Invicton-Labs/go-common/slack/links"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type defaultLoggerHook func(logger Logger) stackerr.Error

type defaultLoggerHookRegistration struct {
	id string
}

func (dlhr defaultLoggerHookRegistration) Close() {
	defaultLoggerHooks.Delete(dlhr.id)
}

var defaultLoggerHooks gensync.Map[string, defaultLoggerHook]

// RegisterDefaultLoggerHook will register a hook function that will be called whenever
// the default logger is updated. This can be used for loggers that wrap the default
// logger in order to update those loggers whenever the default logger gets updated.
func registerDefaultLoggerHook(hook defaultLoggerHook) (defaultLoggerHookRegistration, stackerr.Error) {
	registration := defaultLoggerHookRegistration{
		id: uuid.New().String(),
	}
	// Run it immediately using the existing default logger
	if err := hook(defaultLogger); err != nil {
		return defaultLoggerHookRegistration{}, err
	}
	defaultLoggerHooks.Store(registration.id, hook)
	return registration, nil
}

type DynamicDefaultLogger interface {
	Logger() Logger
	IsDevelopment() bool
	Close()
}
type dynamicDefaultLogger struct {
	lock         sync.Mutex
	registration defaultLoggerHookRegistration
	logger       Logger
}

func (ddl *dynamicDefaultLogger) Logger() Logger {
	ddl.lock.Lock()
	defer ddl.lock.Unlock()
	return ddl.logger
}
func (ddl *dynamicDefaultLogger) Close() {
	ddl.lock.Lock()
	defer ddl.lock.Unlock()
	ddl.registration.Close()
}
func (ddl *dynamicDefaultLogger) IsDevelopment() bool {
	ddl.lock.Lock()
	defer ddl.lock.Unlock()
	return ddl.logger.Config().IsDevelopment
}

func NewDynamicDefaultLogger(loggerConfigFunc func(input NewInput) NewInput) DynamicDefaultLogger {
	ddl := &dynamicDefaultLogger{}
	hook := func(logger Logger) stackerr.Error {
		ddl.lock.Lock()
		defer ddl.lock.Unlock()
		if loggerConfigFunc != nil {
			ddl.logger = New(loggerConfigFunc(logger.Config()))
		} else {
			ddl.logger = New(logger.Config())
		}
		return nil
	}
	ddl.registration, _ = registerDefaultLoggerHook(hook)
	return ddl
}

var defaultLogger Logger
var defaultLoggerLock sync.Mutex

var Debugf func(template string, args ...interface{})
var Infof func(template string, args ...interface{})
var Warnf func(template string, args ...interface{})
var Errorf func(template string, args ...interface{})
var Fatalf func(template string, args ...interface{})
var Panicf func(template string, args ...interface{})

var Debugw func(msg string, keysAndValues ...interface{})
var Infow func(msg string, keysAndValues ...interface{})
var Warnw func(msg string, keysAndValues ...interface{})
var Errorw func(msg string, keysAndValues ...interface{})
var Fatalw func(msg string, keysAndValues ...interface{})
var Panicw func(msg string, keysAndValues ...interface{})

var DebugInterface func(args ...interface{})
var InfoInterface func(args ...interface{})
var WarnInterface func(args ...interface{})
var ErrorInterface func(args ...interface{})
var FatalInterface func(args ...interface{})
var PanicInterface func(args ...interface{})

var Error func(err error)
var Fatal func(err error)
var Panic func(err error)

var With func(args ...interface{}) Logger
var WithOptions func(opts ...zap.Option) Logger
var WithError func(err error) Logger
var WithStackTrace func(stack stackerr.Stack, useAsCaller bool) Logger

// InitDefault will create a new logger with the given settings
// and will set it as the default global logger. This function
// IS NOT thread-safe and cannot be used while other routines
// are using the existing global default logger.
func InitDefault(input NewInput) stackerr.Error {
	defaultLoggerLock.Lock()
	defer defaultLoggerLock.Unlock()

	defaultLogger = New(input)

	Debugf = defaultLogger.Debugf
	Infof = defaultLogger.Infof
	Warnf = defaultLogger.Warnf
	Errorf = defaultLogger.Errorf
	Fatalf = defaultLogger.Fatalf
	Panicf = defaultLogger.Panicf

	Debugw = defaultLogger.Debugw
	Infow = defaultLogger.Infow
	Warnw = defaultLogger.Warnw
	Errorw = defaultLogger.Errorw
	Fatalw = defaultLogger.Fatalw
	Panicw = defaultLogger.Panicw

	DebugInterface = defaultLogger.DebugInterface
	InfoInterface = defaultLogger.InfoInterface
	WarnInterface = defaultLogger.WarnInterface
	ErrorInterface = defaultLogger.ErrorInterface
	FatalInterface = defaultLogger.FatalInterface
	PanicInterface = defaultLogger.PanicInterface

	Error = defaultLogger.Error
	Fatal = defaultLogger.Fatal
	Panic = defaultLogger.Panic

	With = defaultLogger.With
	WithOptions = defaultLogger.WithOptions
	WithError = defaultLogger.WithError
	WithStackTrace = defaultLogger.WithStackTrace

	var err stackerr.Error
	// Run all hooks
	defaultLoggerHooks.Range(func(key string, hook defaultLoggerHook) bool {
		err = hook(defaultLogger)
		return err == nil
	})
	return err
}

// SweetenDefaultLogger will add fields to the default logger.
func SweetenDefaultLogger(fields map[string]any) stackerr.Error {
	input := defaultLogger.Config()
	input.InitialFields = collections.MergeMaps(input.InitialFields, fields)
	return InitDefault(input)
}

// SweetenDefaultLoggerForLambda will add Lambda metadata fields (request ID and logs URL) to the
// default logger, as well as any additional fields in the `fields` parameter.
// If this is not executed within a Lambda function, nothing will be added.
func SweetenDefaultLoggerForLambda(ctx context.Context, fields map[string]any) stackerr.Error {
	input := defaultLogger.Config()
	lambdaMeta, err := iselambda.MetaFromContext(ctx)
	lambdaFields := map[string]any{}
	if err == nil {
		lambdaFields["request_id"] = lambdaMeta.RequestId
		lambdaFields["logs_url"] = zap.Field{
			Type:      zapcore.SkipType,
			Interface: links.NewSlackLink(iselambda.RequestIdLogStreamUrlFromMeta(lambdaMeta), "Log Stream"),
		}
	}
	input.InitialFields = collections.MergeMaps(input.InitialFields, lambdaFields, fields)
	return InitDefault(input)
}

// UnsweetenDefaultLogger will remove fields from the default logger.
func UnsweetenDefaultLogger(fieldKeys []string) stackerr.Error {
	input := defaultLogger.Config()
	needsUpdate := false
	for _, key := range fieldKeys {
		if _, ok := input.InitialFields[key]; ok {
			needsUpdate = true
			delete(input.InitialFields, key)
		}
	}
	if needsUpdate {
		return InitDefault(input)
	}
	return nil
}

func RegisterDefaultWriteHook(key string, hook ZapWriteHook) stackerr.Error {
	defaultLoggerLock.Lock()
	defer defaultLoggerLock.Unlock()
	return defaultLogger.RegisterWriteHook(key, hook)
}

func DeregisterDefaultWriteHook(key string) stackerr.Error {
	defaultLoggerLock.Lock()
	defer defaultLoggerLock.Unlock()
	return defaultLogger.DergisterWriteHook(key)
}
