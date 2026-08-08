package tests

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
	logerrors "github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/types"
)

func TestLogger_File(t *testing.T) {
	tests := []struct {
		name    string
		records []testRecord
	}{
		{
			name: "single record",
			records: []testRecord{
				{
					level:   log.LevelInfo,
					message: "file record",
				},
			},
		},
		{
			name: "multiple records",
			records: []testRecord{
				{
					level:   log.LevelDebug,
					message: "debug record",
				},
				{
					level:   log.LevelInfo,
					message: "info record",
				},
				{
					level:   log.LevelError,
					message: "error record",
				},
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
						Kind:   types.KindFile,
						Format: types.FormatText,
						Level:  types.LevelDebug,
						Path:   path,
					},
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

			for _, record := range tt.records {
				logger.Log(record.level, record.message)
			}

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile() error: %v", err)
			}

			output := string(data)

			for _, record := range tt.records {
				want := record.level.String() + " " + record.message
				if !strings.Contains(output, want) {
					t.Errorf("file output does not contain %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestLogger_File_AppName_DefaultPath(t *testing.T) {
	const appName = "log-e2e"

	logDir := setDefaultLogBase(t, appName)
	logPath := filepath.Join(logDir, appName+".log")

	logger, err := log.NewFileLogger(log.WithAppName(appName))
	if err != nil {
		t.Fatalf("NewFileLogger() error: %v", err)
	}

	logger.Info("default path message")

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", logPath, err)
	}

	if !strings.Contains(string(data), "info default path message") {
		t.Fatalf(
			"log file does not contain expected record\npath: %s\ncontent:\n%s",
			logPath,
			data,
		)
	}
}

func TestLogger_File_AppName_DefaultPath_CannotOpen(t *testing.T) {
	const appName = "log-e2e"

	logDir := setDefaultLogBase(t, appName)

	// Put a regular file where the logger expects its log directory.
	if err := os.MkdirAll(filepath.Dir(logDir), 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	if err := os.WriteFile(logDir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cfg := log.Config{
		AppName: appName,
		Sinks: []log.SinkConfig{
			{
				Kind: log.KindFile,
			},
		},
	}

	if err := cfg.ApplyDefaults(); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	_, err := log.NewLogger(&cfg)
	if err == nil {
		t.Fatal("NewLogger() expected error, got nil")
	}

	if !errors.Is(err, logerrors.ErrCannotOpenFileWriter) {
		t.Errorf(
			"NewLogger() error = %v, want errors.Is(err, ErrCannotOpenFileWriter)",
			err,
		)
	}
}

func setDefaultLogBase(t *testing.T, appName string) string {
	t.Helper()

	root := t.TempDir()

	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", root)

		return filepath.Join(
			root,
			"Library",
			"Application Support",
			appName,
		)

	case "windows":
		t.Setenv("USERPROFILE", root)

		appData := filepath.Join(root, "AppData", "Roaming")
		t.Setenv("APPDATA", appData)

		return filepath.Join(appData, appName)

	default:
		t.Setenv("HOME", root)

		xdgDataHome := filepath.Join(root, "data")
		t.Setenv("XDG_DATA_HOME", xdgDataHome)

		return filepath.Join(xdgDataHome, appName)
	}
}
