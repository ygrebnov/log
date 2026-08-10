package log

import (
	"context"
	"errors"
	"strconv"

	"github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/internal/core"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

type Level = types.Level
type Field = types.Field
type Kind = types.Kind
type Format = types.Format
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
)

func String(key keys.Key, value string) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

func Int(key keys.Key, value int) Field {
	return Field{
		Key:   key,
		Value: strconv.Itoa(value),
	}
}

func Bool(key keys.Key, value bool) Field {
	return Field{
		Key:   key,
		Value: strconv.FormatBool(value),
	}
}

func Err(key keys.Key, err error) Field {
	if err == nil {
		return Field{Key: key}
	}

	return Field{
		Key:   key,
		Value: err.Error(),
	}
}

type Logger struct {
	handler *core.Handler
}

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

	l.handler.Log(level, msg, fields...)
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

func NewSilentLogger() (*Logger, error) {
	return NewLogger(nil)
}

type loggerConfig struct {
	level   Level
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

func NewStdoutLogger(opts ...Option) (*Logger, error) {
	return newSingleSinkLogger(KindStdOut, opts...)
}

func NewStderrLogger(opts ...Option) (*Logger, error) {
	return newSingleSinkLogger(KindStdErr, opts...)
}

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
				Kind:  kind,
				Level: o.level,
				Path:  o.path,
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
