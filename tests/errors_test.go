package tests

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
	logerrors "github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/types"
)

func TestLogger_Errors(t *testing.T) {
	t.Run("cannot create log directory", func(t *testing.T) {
		dir := t.TempDir()

		parent := filepath.Join(dir, "not-a-directory")
		if err := os.WriteFile(parent, []byte("file"), 0o600); err != nil {
			t.Fatalf("WriteFile() error: %v", err)
		}

		cfg := &config.Config{
			AppName: "log-e2e",
			Sinks: []config.SinkConfig{
				{
					Kind:   types.KindFile,
					Format: types.FormatText,
					Level:  types.LevelInfo,
					Path:   filepath.Join(parent, "app.log"),
				},
			},
		}
		err := cfg.ApplyDefaults()
		if err != nil {
			t.Fatalf("ApplyDefaults() error: %v", err)
		}

		_, err = log.NewLogger(cfg)
		if err == nil {
			t.Fatal("NewLogger() expected error, got nil")
		}

		if !errors.Is(err, logerrors.ErrCannotOpenFileWriter) {
			t.Fatalf(
				"NewLogger() error = %v, want errors.Is(..., ErrCannotOpenFileWriter)",
				err,
			)
		}

		if !errors.Is(err, logerrors.ErrCannotOpenFileWriter) {
			t.Fatalf(
				"NewLogger() error = %v, want errors.Is(..., ErrCannotAccessLogFilePath)",
				err,
			)
		}
	})

	t.Run("invalid level text", func(t *testing.T) {
		var level log.Level

		err := level.UnmarshalText([]byte("invalid"))
		if err == nil {
			t.Fatal("UnmarshalText() expected error, got nil")
		}

		if !errors.Is(err, logerrors.ErrInvalidLogLevel) {
			t.Fatalf(
				"UnmarshalText() error = %v, want errors.Is(..., ErrInvalidLogLevel)",
				err,
			)
		}
	})

	t.Run("invalid level JSON", func(t *testing.T) {
		var level log.Level

		err := level.UnmarshalJSON([]byte(`"invalid"`))
		if err == nil {
			t.Fatal("UnmarshalJSON() expected error, got nil")
		}

		if !errors.Is(err, logerrors.ErrInvalidLogLevel) {
			t.Fatalf(
				"UnmarshalJSON() error = %v, want errors.Is(..., ErrInvalidLogLevel)",
				err,
			)
		}
	})
}
