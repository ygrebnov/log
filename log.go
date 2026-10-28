package log

import (
	"context"
	"errors"
	"time"

	"github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/internal/core"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

type Record = types.Record
type Field = types.Field
type Level = types.Level
type Kind = types.Kind
type Format = types.Format
type TimestampFormat = types.TimestampFormat
type Config = config.Config
type SinkConfig = config.SinkConfig

const (
	LevelTrace = types.LevelTrace
	LevelDebug = types.LevelDebug
	LevelInfo  = types.LevelInfo
	LevelWarn  = types.LevelWarn
	LevelError = types.LevelError
	LevelFatal = types.LevelFatal

	KindStdOut = types.KindStdOut
	KindStdErr = types.KindStdErr
	KindFile   = types.KindFile

	FormatJSON = types.FormatJSON
	FormatText = types.FormatText

	TimestampFormatRFC3339Nano = types.TimestampFormatRFC3339Nano
	TimestampFormatRFC3339     = types.TimestampFormatRFC3339
)

var NewRecord = types.NewRecord

func String(key keys.Key, value string) Field {
	return types.NewStringField(key, value)
}

func Int(key keys.Key, value int) Field {
	return types.NewIntField(key, value)
}

func Int64(key keys.Key, value int64) Field {
	return types.NewInt64Field(key, value)
}

func Bool(key keys.Key, value bool) Field {
	return types.NewBoolField(key, value)
}

func Err(key keys.Key, err error) Field {
	return types.NewErrField(key, err)
}

type Logger struct {
	handler *core.Handler
}

// NewLogger creates a new logger with the provided configuration.
// If cfg is nil, a silent logger without sinks will be created.
func NewLogger(cfg *config.Config) (*Logger, error) {
	return NewLoggerCtx(context.Background(), cfg)
}

func NewLoggerCtx(ctx context.Context, cfg *config.Config) (l *Logger, e error) {
	if cfg == nil {
		cfg = &config.Config{}
	}

	if err := cfg.Validate(ctx); err != nil {
		return nil, err
	}

	cl := newCloser()
	defer func() {
		if e != nil {
			e = errors.Join(e, cl.Close())
		}
	}()

	sinks := make([]*core.Sink, 0, len(cfg.Sinks))
	for i := range cfg.Sinks {
		sink, err := core.NewSink(ctx, &cfg.Sinks[i], cfg.AppName)
		if err != nil {
			e = err
			return
		}

		sinks = append(sinks, sink)
		cl.Add(sink)
	}

	handler := core.NewHandler(sinks...)

	return &Logger{
		handler: handler,
	}, nil
}

func (l *Logger) Log(level Level, msg string, fields ...Field) {
	if l == nil || l.handler == nil {
		return
	}

	record := NewRecord(time.Now(), level, msg, fields...)

	// copy fields to protect against modification after the call, for example:
	// logger.Info("request", fields...)
	// fields[0] = log.String(keys.New("status"), "completed")
	l.handler.LogRecord(record, true)
}

func (l *Logger) LogRecord(record Record) {
	if l == nil || l.handler == nil {
		return
	}

	l.handler.LogRecord(record, true)
}

func (l *Logger) Trace(msg string, fields ...Field) {
	l.Log(LevelTrace, msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.Log(LevelDebug, msg, fields...)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.Log(LevelInfo, msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.Log(LevelWarn, msg, fields...)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.Log(LevelError, msg, fields...)
}

// Fatal adds a log record at the fatal level, but does not call os.Exit(1).
func (l *Logger) Fatal(msg string, fields ...Field) {
	l.Log(LevelFatal, msg, fields...)
}

func (l *Logger) Close() error {
	if l == nil || l.handler == nil {
		return nil
	}

	return l.handler.Close()
}

// With returns a new logger with the provided fields added to all log records.
// Loggers created using With share the parent’s sinks.
// Closing any of them closes the shared logging resources.
func (l *Logger) With(fields ...Field) *Logger {
	if l == nil || l.handler == nil {
		return l
	}

	newHandler := l.handler.With(fields...)

	return &Logger{
		handler: newHandler,
	}
}

func NewSilentLogger() (*Logger, error) {
	return NewLogger(nil)
}

type loggerConfig struct {
	level   Level
	format  Format
	path    string
	appName string
}

type Option func(*loggerConfig)

func WithLevel(level Level) Option {
	return func(config *loggerConfig) {
		config.level = level
	}
}

func WithPath(path string) Option {
	return func(config *loggerConfig) {
		config.path = path
	}
}

func WithAppName(appName string) Option {
	return func(config *loggerConfig) {
		config.appName = appName
	}
}

func WithFormat(format Format) Option {
	return func(config *loggerConfig) {
		config.format = format
	}
}

// NewStdoutLogger creates a new logger that writes to stdout.
// By default, logger uses the stdout sink default configuration.
// Use WithLevel or WithFormat options to customize it.
func NewStdoutLogger(opts ...Option) (*Logger, error) {
	return newSingleSinkLogger(KindStdOut, opts...)
}

// NewStderrLogger creates a new logger that writes to stderr.
// By default, logger uses the stderr sink default configuration.
// Use WithLevel or WithFormat options to customize it.
func NewStderrLogger(opts ...Option) (*Logger, error) {
	return newSingleSinkLogger(KindStdErr, opts...)
}

// NewFileLogger creates a new logger that writes to a file.
// By default, logger uses the file sink default configuration.
// Use WithLevel, WithFormat, WithAppName, or WithPath options to customize it.
func NewFileLogger(opts ...Option) (*Logger, error) {
	return newSingleSinkLogger(KindFile, opts...)
}

func newSingleSinkLogger(kind Kind, opts ...Option) (*Logger, error) {
	o := &loggerConfig{}

	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	cfg := &Config{
		AppName: o.appName,
		Sinks: []SinkConfig{
			{
				Kind:   kind,
				Level:  o.level,
				Format: o.format,
				Path:   o.path,
			},
		},
	}

	if err := cfg.ApplyDefaults(); err != nil {
		return nil, err
	}

	return NewLogger(cfg)
}

type loggerCtxKey struct{}

// ToCtx returns a new context containing logger.
func ToCtx(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// FromCtx returns the logger stored in ctx, or nil if none is present.
func FromCtx(ctx context.Context) *Logger {
	logger, _ := ctx.Value(loggerCtxKey{}).(*Logger)
	return logger
}
