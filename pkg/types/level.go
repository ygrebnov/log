package types

import (
	"encoding/json"
	"strings"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/log/pkg/errors"
	keyspkg "github.com/ygrebnov/log/pkg/keys"
)

// Level defines log level.
//
// Values are aligned with slog.Level:
//   - Debug: -4
//   - Info:   0
//   - Warn:   4
//   - Error:  8
//
// Trace and Fatal extend the scale in the same increments.
type Level int

const (
	LevelTrace Level = -8
	LevelDebug Level = -4
	LevelInfo  Level = 0
	LevelWarn  Level = 4
	LevelError Level = 8
	LevelFatal Level = 12
)

var levelToString = map[Level]string{
	LevelTrace: "trace",
	LevelDebug: "debug",
	LevelInfo:  "info",
	LevelWarn:  "warn",
	LevelError: "error",
	LevelFatal: "fatal",
}

var stringToLevel = map[string]Level{
	"trace":   LevelTrace,
	"debug":   LevelDebug,
	"info":    LevelInfo,
	"warn":    LevelWarn,
	"warning": LevelWarn,
	"error":   LevelError,
	"fatal":   LevelFatal,
}

func (l Level) String() string {
	value, ok := levelToString[l]
	if !ok {
		return "unknown"
	}

	return value
}

func (l Level) MarshalText() ([]byte, error) {
	value, ok := levelToString[l]
	if !ok {
		return nil, errorc.With(errors.ErrInvalidLogLevel, errorc.Int(keyspkg.LogLevel, int(l)))
	}

	return []byte(value), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	value := strings.ToLower(strings.TrimSpace(string(text)))

	level, ok := stringToLevel[value]
	if !ok {
		return errorc.With(errors.ErrInvalidLogLevel, errorc.String(keyspkg.LogLevel, value))
	}

	*l = level

	return nil
}

func (l Level) MarshalJSON() ([]byte, error) {
	text, err := l.MarshalText()
	if err != nil {
		return nil, err
	}

	return json.Marshal(string(text))
}

func (l *Level) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return errorc.With(errors.ErrInvalidLogLevel, errorc.Error(keyspkg.Cause, err))
	}

	return l.UnmarshalText([]byte(value))
}
