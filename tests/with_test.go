package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ygrebnov/keys"
	log "github.com/ygrebnov/log"
)

func TestLogger_With(t *testing.T) {
	tests := []struct {
		name    string
		log     func(base *log.Logger)
		want    []string
		wantNot []string
	}{
		{
			name: "single With",
			log: func(base *log.Logger) {
				logger := base.With(
					log.String(keys.New("service"), "api"),
				)

				logger.Info("request received")
			},
			want: []string{
				`info request received`,
				`service="api"`,
			},
		},
		{
			name: "base logger is not modified",
			log: func(base *log.Logger) {
				_ = base.With(
					log.String(keys.New("service"), "api"),
				)

				base.Info("base record")
			},
			want: []string{
				`info base record`,
			},
			wantNot: []string{
				`service="api"`,
			},
		},
		{
			name: "chained With",
			log: func(base *log.Logger) {
				logger := base.
					With(
						log.String(keys.New("service"), "api"),
					).
					With(
						log.String(keys.New("request_id"), "123"),
					)

				logger.Info("request completed")
			},
			want: []string{
				`info request completed`,
				`service="api"`,
				`request_id="123"`,
			},
		},
		{
			name: "record fields follow bound fields",
			log: func(base *log.Logger) {
				logger := base.With(
					log.String(keys.New("service"), "api"),
					log.String(keys.New("source"), "bound"),
				)

				logger.Info(
					"request completed",
					log.String(keys.New("source"), "record"),
					log.Int(keys.New("status"), 200),
				)
			},
			want: []string{
				`service="api" source="bound" source="record" status="200"`,
			},
		},
		{
			name: "different derived loggers are independent",
			log: func(base *log.Logger) {
				first := base.With(
					log.String(keys.New("component"), "first"),
				)

				second := base.With(
					log.String(keys.New("component"), "second"),
				)

				first.Info("first record")
				second.Info("second record")
			},
			want: []string{
				`info first record component="first"`,
				`info second record component="second"`,
			},
			wantNot: []string{
				`info first record component="second"`,
				`info second record component="first"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "with.log")

			logger, err := log.NewFileLogger(
				log.WithPath(path),
				log.WithLevel(log.LevelTrace),
			)
			if err != nil {
				t.Fatalf("NewFileLogger() error: %v", err)
			}

			tt.log(logger)

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error: %v", err)
			}

			output := string(data)

			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf(
						"output does not contain %q:\n%s",
						want,
						output,
					)
				}
			}

			for _, wantNot := range tt.wantNot {
				if strings.Contains(output, wantNot) {
					t.Errorf(
						"output unexpectedly contains %q:\n%s",
						wantNot,
						output,
					)
				}
			}
		})
	}
}

func TestLogger_With_SharedLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "with-close.log")

	base, err := log.NewFileLogger(
		log.WithPath(path),
	)
	if err != nil {
		t.Fatalf("NewFileLogger() error: %v", err)
	}

	derived := base.With(
		log.String(keys.New("service"), "api"),
	)

	derived.Info("before close")

	// A derived Logger shares the same Handler lifecycle.
	if err := derived.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Must be safe and ignored after the shared handler is closed.
	base.Info("after close")
	derived.Info("derived after close")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	output := string(data)

	if !strings.Contains(
		output,
		`info before close service="api"`,
	) {
		t.Errorf(
			"output does not contain expected record:\n%s",
			output,
		)
	}

	if strings.Contains(output, "after close") {
		t.Errorf(
			"output contains record written after Close():\n%s",
			output,
		)
	}
}

func TestLogger_With_Nil(t *testing.T) {
	var logger *log.Logger

	got := logger.With(
		log.String(keys.New("service"), "api"),
	)

	if got != nil {
		t.Fatalf("With() = %v, want nil", got)
	}
}
