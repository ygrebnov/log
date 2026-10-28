package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

func TestLogger_StdoutAndFile(t *testing.T) {
	tests := []struct {
		name string

		stdoutLevel log.Level
		fileLevel   log.Level

		records []testRecord

		wantStdout    []string
		wantNotStdout []string
		wantFile      []string
		wantNotFile   []string
	}{
		{
			name:        "same level",
			stdoutLevel: log.LevelInfo,
			fileLevel:   log.LevelInfo,
			records: []testRecord{
				{level: log.LevelInfo, message: "first"},
				{level: log.LevelWarn, message: "second"},
			},
			wantStdout: []string{
				"INFO first",
				"WARN second",
			},
			wantFile: []string{
				"INFO first",
				"WARN second",
			},
		},
		{
			name:        "file is more verbose",
			stdoutLevel: log.LevelInfo,
			fileLevel:   log.LevelDebug,
			records: []testRecord{
				{level: log.LevelDebug, message: "debug only in file"},
				{level: log.LevelInfo, message: "info everywhere"},
			},
			wantStdout: []string{
				"INFO info everywhere",
			},
			wantNotStdout: []string{
				"DEBUG debug only in file",
			},
			wantFile: []string{
				"DEBUG debug only in file",
				"INFO info everywhere",
			},
		},
		{
			name:        "stdout is more verbose",
			stdoutLevel: log.LevelDebug,
			fileLevel:   log.LevelError,
			records: []testRecord{
				{level: log.LevelDebug, message: "debug stdout"},
				{level: log.LevelInfo, message: "info stdout"},
				{level: log.LevelError, message: "error everywhere"},
			},
			wantStdout: []string{
				"DEBUG debug stdout",
				"INFO info stdout",
				"ERROR error everywhere",
			},
			wantFile: []string{
				"ERROR error everywhere",
			},
			wantNotFile: []string{
				"DEBUG debug stdout",
				"INFO info stdout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "app.log")

			cfg := &config.Config{
				AppName: "log-e2e",
				Sinks: []config.SinkConfig{
					{
						Kind:   types.KindStdOut,
						Format: types.FormatText,
						Level:  tt.stdoutLevel,
					},
					{
						Kind:   types.KindFile,
						Format: types.FormatText,
						Level:  tt.fileLevel,
						Path:   path,
					},
				},
			}
			err := cfg.ApplyDefaults()
			if err != nil {
				t.Fatalf("unexpected ApplyDefaults error: %v", err)
			}

			stdout := captureStdout(t, func() {
				logger, err := log.NewLogger(cfg)
				if err != nil {
					t.Fatalf("NewLogger() error: %v", err)
				}

				for _, record := range tt.records {
					logger.Log(record.level, record.message)
				}

				if err := logger.Close(); err != nil {
					t.Fatalf("Close() error: %v", err)
				}
			})

			fileData, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error: %v", err)
			}

			fileOutput := string(fileData)

			assertContainsAll(t, stdout, tt.wantStdout)
			assertContainsNone(t, stdout, tt.wantNotStdout)

			assertContainsAll(t, fileOutput, tt.wantFile)
			assertContainsNone(t, fileOutput, tt.wantNotFile)
		})
	}
}
