package tests

import (
	"fmt"
	"strings"
	"testing"

	log "github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/keys"
)

func TestLogger_SpecificAndNegativeScenarios(t *testing.T) {
	t.Run("log after close is ignored", func(t *testing.T) {
		output := captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelTrace)

			logger.Log(log.LevelInfo, "before close")

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}

			logger.Log(log.LevelInfo, "after close")
		})

		assertContainsAll(t, output, []string{
			"info before close",
		})

		assertContainsNone(t, output, []string{
			"after close",
		})
	})

	t.Run("close is idempotent", func(t *testing.T) {
		output := captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelInfo)

			logger.Log(log.LevelInfo, "record")

			if err := logger.Close(); err != nil {
				t.Fatalf("first Close() error: %v", err)
			}

			if err := logger.Close(); err != nil {
				t.Fatalf("second Close() error: %v", err)
			}
		})

		if got := countMessage(output, "info record"); got != 1 {
			t.Fatalf("record count = %d, want 1:\n%s", got, output)
		}
	})

	t.Run("nil error field", func(t *testing.T) {
		output := captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelInfo)

			logger.Log(
				log.LevelInfo,
				"nil error",
				log.Err(keys.Cause, nil),
			)

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})

		assertContainsAll(t, output, []string{
			`info nil error`,
			`cause=""`,
		})
	})

	t.Run("all levels", func(t *testing.T) {
		output := captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelTrace)

			levels := []log.Level{
				log.LevelTrace,
				log.LevelDebug,
				log.LevelInfo,
				log.LevelWarn,
				log.LevelError,
				log.LevelFatal,
			}

			for _, level := range levels {
				logger.Log(level, level.String()+" message")
			}

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})

		for _, level := range []log.Level{
			log.LevelTrace,
			log.LevelDebug,
			log.LevelInfo,
			log.LevelWarn,
			log.LevelError,
			log.LevelFatal,
		} {
			want := fmt.Sprintf("%s %s message", level, level)
			if !strings.Contains(output, want) {
				t.Errorf("output does not contain %q:\n%s", want, output)
			}
		}
	})
}
