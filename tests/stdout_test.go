package tests

import (
	"errors"
	"strings"
	"testing"

	keyslib "github.com/ygrebnov/keys"
	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/keys"
)

func TestLogger_Stdout(t *testing.T) {
	tests := []struct {
		name    string
		log     func(*log.Logger)
		want    []string
		wantNot []string
	}{
		{
			name: "message",
			log: func(logger *log.Logger) {
				logger.Log(log.LevelInfo, "application started")
			},
			want: []string{
				`info application started`,
			},
		},
		{
			name: "fields",
			log: func(logger *log.Logger) {
				logger.Log(
					log.LevelWarn,
					"request completed",
					log.String(keyslib.New("log.message"), "hello"),
					log.Int(keyslib.New("log.level"), 42),
					log.Bool(keyslib.New("enabled"), true),
				)
			},
			want: []string{
				`warn request completed`,
				`log.message="hello"`,
				`log.level="42"`,
				`enabled="true"`,
			},
		},
		{
			name: "error field",
			log: func(logger *log.Logger) {
				logger.Log(
					log.LevelError,
					"operation failed",
					log.Err(keys.Cause, errors.New("boom")),
				)
			},
			want: []string{
				`error operation failed`,
				`cause="boom"`,
			},
		},
		{
			name: "quoted field",
			log: func(logger *log.Logger) {
				logger.Log(
					log.LevelInfo,
					"quoted",
					log.String(keyslib.New("log.message"), "hello \"world\"\nnext"),
				)
			},
			want: []string{
				`info quoted`,
				`log.message="hello \"world\"\nnext"`,
			},
		},
		{
			name: "below sink level is ignored",
			log: func(logger *log.Logger) {
				logger.Log(log.LevelDebug, "debug message")
				logger.Log(log.LevelInfo, "info message")
				logger.Log(log.LevelWarn, "warn message")
			},
			want: []string{
				`info info message`,
				`warn warn message`,
			},
			wantNot: []string{
				`debug message`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				logger, err := log.NewStdoutLogger()
				if err != nil {
					t.Fatalf("NewStdoutLogger() error: %v", err)
				}

				tt.log(logger)

				if err := logger.Close(); err != nil {
					t.Fatalf("Close() error: %v", err)
				}
			})

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("output does not contain %q:\n%s", want, output)
				}
			}

			for _, wantNot := range tt.wantNot {
				if strings.Contains(output, wantNot) {
					t.Errorf("output unexpectedly contains %q:\n%s", wantNot, output)
				}
			}
		})
	}
}
