package tests

import (
	"strings"
	"testing"

	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

func TestLogger_Stderr(t *testing.T) {
	tests := []struct {
		name    string
		level   log.Level
		message string
	}{
		{
			name:    "info",
			level:   log.LevelInfo,
			message: "hello stderr",
		},
		{
			name:    "error",
			level:   log.LevelError,
			message: "stderr error",
		},
		{
			name:    "fatal",
			level:   log.LevelFatal,
			message: "fatal message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(t, func() {
				cfg := &config.Config{
					Sinks: []config.SinkConfig{
						textSink(types.KindStdErr, tt.level),
					},
				}
				err := cfg.ApplyDefaults()
				if err != nil {
					t.Fatalf("unexpected ApplyDefaults error: %v", err)
				}

				logger, err := log.NewLogger(cfg)
				if err != nil {
					t.Fatalf("NewLogger() error: %v", err)
				}

				logger.Log(tt.level, tt.message)

				if err := logger.Close(); err != nil {
					t.Fatalf("Close() error: %v", err)
				}
			})

			want := tt.level.String() + " " + tt.message
			if !strings.Contains(output, want) {
				t.Fatalf("output does not contain %q:\n%s", want, output)
			}
		})
	}
}
