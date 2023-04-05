package log

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Invicton-Labs/go-common/collections"
	"github.com/Invicton-Labs/go-stackerr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
	Panicf(template string, args ...interface{})

	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
	Panicw(msg string, keysAndValues ...interface{})

	DebugInterface(args ...interface{})
	InfoInterface(args ...interface{})
	WarnInterface(args ...interface{})
	ErrorInterface(args ...interface{})
	FatalInterface(args ...interface{})
	PanicInterface(args ...interface{})

	Error(err error)
	Panic(err error)
	Fatal(err error)

	With(args ...interface{}) Logger
	WithOptions(opts ...zap.Option) Logger
	WithError(err error) Logger
	WithStackTrace(stack stackerr.Stack, useAsCaller bool) Logger

	// WithAdditionalSkippedFrames will return a new logger that skips additional
	// frames when finding the caller and the stack trace.
	WithAdditionalSkippedFrames(skippedFrames int) Logger

	// RegisterWriteHook will register a function hook that will be called
	// for each log write.
	RegisterWriteHook(key string, hook ZapWriteHook) stackerr.Error

	// DergisterWriteHook will deregister a hook that was registered with RegisterWriteHook
	DergisterWriteHook(key string) stackerr.Error

	// Config gets the config values that can be used to re-create this logger
	Config() NewInput

	// Clone returns a copy of the logger
	Clone() Logger
}

type logger struct {
	*zap.SugaredLogger
	config NewInput
}

func (l logger) Clone() Logger {
	return logger{
		SugaredLogger: l.SugaredLogger.With(),
		config:        l.config.Clone(),
	}
}

func (l logger) Config() NewInput {
	return l.config.Clone()
}
func (l logger) RegisterWriteHook(key string, hook ZapWriteHook) stackerr.Error {
	if _, ok := l.config.WriteHooks[key]; ok {
		return stackerr.Errorf("Write hook key `%s` is already registered", key)
	}
	l.config.WriteHooks[key] = hook
	return nil
}
func (l logger) DergisterWriteHook(key string) stackerr.Error {
	if _, ok := l.config.WriteHooks[key]; !ok {
		return stackerr.Errorf("Write hook key `%s` is not registered", key)
	}
	delete(l.config.WriteHooks, key)
	return nil
}

func (l logger) getLoggerWithErrAndFields(err error, additionalFrameSkips int) (logger, []any) {
	// Convert it to a stackerr if it isn't one already
	serr := stackerr.WrapWithFrameSkipsWithoutExtraStack(err, 1+additionalFrameSkips)
	// Add the stacktraces to the stacktraces fields
	// We add and extra frame skip to the error so it doesn't capture this function. We
	// don't need the extra frame skip for the logger because we're not logging from within this function.
	l = l.withErrorSkipFrames(err, 1+additionalFrameSkips, false).(logger).WithOptions(zap.AddCallerSkip(additionalFrameSkips)).(logger)
	kvp := make([]any, 0, 2*len(serr.Fields()))
	for k, v := range serr.Fields() {
		kvp = append(kvp, k, v)
	}
	return l, kvp
}

// Error will add the error fields as log fields, will add a stack trace if one isn't already
// set in the error, and will then log it at the Error level.
func (l logger) Error(err error) {
	l, f := l.getLoggerWithErrAndFields(err, 1)
	l.Errorw(err.Error(), f...)
}

// Panic will add the error fields as log fields, will add a stack trace if one isn't already
// set in the error, and will then log it at the Panic level.
func (l logger) Panic(err error) {
	l, f := l.getLoggerWithErrAndFields(err, 1)
	l.Panicw(err.Error(), f...)
}

// Fatal will add the error fields as log fields, will add a stack trace if one isn't already
// set in the error, and will then log it at the Fatal level.
func (l logger) Fatal(err error) {
	l, f := l.getLoggerWithErrAndFields(err, 1)
	l.Fatalw(err.Error(), f...)
}

func (l logger) DebugInterface(args ...any) {
	l.SugaredLogger.WithOptions(zap.AddCallerSkip(1)).Debug(args...)
}
func (l logger) InfoInterface(args ...any) {
	l.SugaredLogger.WithOptions(zap.AddCallerSkip(1)).Info(args...)
}
func (l logger) WarnInterface(args ...any) {
	l.SugaredLogger.WithOptions(zap.AddCallerSkip(1)).Warn(args...)
}
func (l logger) ErrorInterface(args ...any) {
	l.SugaredLogger.WithOptions(zap.AddCallerSkip(1)).Error(args...)
}
func (l logger) PanicInterface(args ...any) {
	l.SugaredLogger.WithOptions(zap.AddCallerSkip(1)).Panic(args...)
}
func (l logger) FatalInterface(args ...any) {
	l.SugaredLogger.WithOptions(zap.AddCallerSkip(1)).Fatal(args...)
}

func (l logger) With(args ...interface{}) Logger {
	return logger{l.SugaredLogger.With(args...), l.config.Clone()}
}

func (l logger) WithOptions(opts ...zap.Option) Logger {
	return logger{l.SugaredLogger.WithOptions(opts...), l.config.Clone()}
}

// withErrorSkipFrames will return a new logger with the error added. If addErrField is true,
// the error will be added to the "errs" field. Otherwise, only the stacktraces will be added to the
// "stacktraces" field and the error message will be discarded.
func (l logger) withErrorSkipFrames(err error, skippedFrames int, addErrField bool) Logger {
	if err == nil {
		err = fmt.Errorf("")
		addErrField = false
	}
	if serr, ok := err.(stackerr.Error); ok || errors.As(err, &serr) {
		if addErrField {
			return l.With(zapcore.Field{
				Type:      zapcore.SkipType,
				Interface: serr,
			})
		}
		stacks := serr.Stacks()
		stackFields := make([]any, 0, len(stacks))
		for _, stack := range stacks {
			stackFields = append(stackFields, zapcore.Field{
				Type: zapcore.SkipType,
				Interface: stackTrace{
					stack:       stack,
					useAsCaller: false,
				},
			})
		}
		return l.With(stackFields...)
	}

	return l.withErrorSkipFrames(stackerr.WrapWithFrameSkips(err, 1+skippedFrames), 1+skippedFrames, addErrField)
}

func (l logger) WithError(err error) Logger {
	return l.withErrorSkipFrames(err, 1, true)
}

func (l logger) WithAdditionalSkippedFrames(skippedFrames int) Logger {
	return l.WithOptions(zap.AddCallerSkip(skippedFrames))
}

func (l logger) WithStackTrace(stack stackerr.Stack, useAsCaller bool) Logger {
	return l.With(zapcore.Field{
		Type: zapcore.SkipType,
		Interface: stackTrace{
			stack:       stack,
			useAsCaller: useAsCaller,
		},
	})
}

type NewInput struct {
	Name          string
	Level         zapcore.Level
	IsDevelopment bool
	WriteHooks    map[string]ZapWriteHook
	InitialFields map[string]any
	SkippedFrames int
}

func (ni *NewInput) Clone() NewInput {
	return NewInput{
		Name:          ni.Name,
		Level:         ni.Level,
		IsDevelopment: ni.IsDevelopment,
		WriteHooks:    collections.CopyMap(ni.WriteHooks),
		InitialFields: collections.CopyMap(ni.InitialFields),
		SkippedFrames: ni.SkippedFrames,
	}
}

func New(input NewInput) Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktraces",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder

	if input.IsDevelopment {
		// If it's development mode, modify some settings
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	sink, closeOut, err := zap.Open("stdout")
	if err != nil {
		panic(err)
	}
	errSink, _, err := zap.Open("stderr")
	if err != nil {
		closeOut()
		panic(err)
	}

	buildOpts := []zap.Option{
		zap.ErrorOutput(errSink),
	}

	if input.IsDevelopment {
		buildOpts = append(buildOpts, zap.Development())
	}

	// Add the caller field
	buildOpts = append(buildOpts, zap.AddCaller())

	// Add the stacktraces
	buildOpts = append(buildOpts, zap.AddStacktrace(zap.WarnLevel))

	if !input.IsDevelopment {
		samplingCfg := &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
		buildOpts = append(buildOpts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			var samplerOpts []zapcore.SamplerOption
			if samplingCfg.Hook != nil {
				samplerOpts = append(samplerOpts, zapcore.SamplerHook(samplingCfg.Hook))
			}
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				samplingCfg.Initial,
				samplingCfg.Thereafter,
				samplerOpts...,
			)
		}))
	}

	if input.WriteHooks == nil {
		input.WriteHooks = map[string]ZapWriteHook{}
	}
	if input.InitialFields == nil {
		input.InitialFields = map[string]any{}
	}

	// Add any initial field as a build option
	if len(input.InitialFields) > 0 {
		fs := make([]zap.Field, 0, len(input.InitialFields))
		keys := make([]string, 0, len(input.InitialFields))
		for k := range input.InitialFields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if f, ok := input.InitialFields[k].(zap.Field); ok {
				f.Key = k
				fs = append(fs, f)
			} else {
				fs = append(fs, zap.Any(k, input.InitialFields[k]))
			}
		}
		buildOpts = append(buildOpts, zap.Fields(fs...))
	}

	if input.SkippedFrames != 0 {
		buildOpts = append(buildOpts, zap.AddCallerSkip(input.SkippedFrames))
	}

	levelEnabler := zap.NewAtomicLevelAt(input.Level)

	if input.WriteHooks == nil {
		input.WriteHooks = map[string]ZapWriteHook{}
	} else {
		input.WriteHooks = collections.CopyMap(input.WriteHooks)
	}

	zapLogger := zap.New(
		&core{
			LevelEnabler:  levelEnabler,
			name:          input.Name,
			enc:           encoder,
			out:           sink,
			isJson:        !input.IsDevelopment,
			fields:        map[string]zapcore.Field{},
			getWriteHooks: func() map[string]ZapWriteHook { return input.WriteHooks },
		},
		buildOpts...,
	)

	return &logger{zapLogger.Sugar(), input}
}
