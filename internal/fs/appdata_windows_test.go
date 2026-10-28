package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ygrebnov/log/pkg/config"
)

func TestWindowsAppDataPathValidation(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	appName := "appdata-test"

	tests := []struct {
		name    string
		appData string
		wantDir string
		wantErr bool
	}{
		{
			name:    "valid path inside home",
			appData: filepath.Join(home, "AppData", "Roaming"),
			wantDir: filepath.Join(home, "AppData", "Roaming", appName),
		},
		{
			name:    "custom path inside home",
			appData: filepath.Join(home, "CustomData"),
			wantDir: filepath.Join(home, "CustomData", appName),
		},
		{
			name:    "empty APPDATA",
			appData: "",
			wantDir: filepath.Join(home, "AppData", "Roaming", appName),
		},
		{
			name:    "relative APPDATA",
			appData: filepath.Join("relative", "AppData"),
			wantErr: true,
		},
		{
			name:    "APPDATA outside home",
			appData: filepath.Join(filepath.Dir(home), "outside", "AppData"),
			wantErr: true,
		},
		{
			name:    "parent traversal outside home",
			appData: filepath.Join(home, "..", "outside", "AppData"),
			wantErr: true,
		},
		{
			name:    "similar home path prefix",
			appData: home + "-other",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("APPDATA", tt.appData)

			dir, err := getLogDir(appName)

			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"getLogDir() = %q, expected error",
						dir,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("getLogDir() failed: %v", err)
			}

			if filepath.Clean(dir) != filepath.Clean(tt.wantDir) {
				t.Fatalf(
					"getLogDir() = %q, want %q",
					dir,
					tt.wantDir,
				)
			}
		})
	}
}

func TestWindowsAppDataUnset(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("APPDATA", "")
	if err := os.Unsetenv("APPDATA"); err != nil {
		t.Fatal(err)
	}

	appName := "appdata-unset-test"

	dir, err := getLogDir(appName)
	if err != nil {
		t.Fatalf("getLogDir() failed: %v", err)
	}

	want := filepath.Join(home, "AppData", "Roaming", appName)

	if filepath.Clean(dir) != filepath.Clean(want) {
		t.Fatalf(
			"getLogDir() = %q, want %q",
			dir,
			want,
		)
	}
}

func TestWindowsAppDataOpenWriter(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	// Create a temporary APPDATA base under the real user home.
	// This satisfies the logger's home-directory restriction.
	base, err := os.MkdirTemp(home, "log-appdata-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(base); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	t.Setenv("APPDATA", base)

	appName := "appdata-integration-test"

	writer, closer, err := OpenWriter(
		&config.SinkConfig{},
		appName,
	)
	if err != nil {
		t.Fatalf("OpenWriter() failed: %v", err)
	}

	if _, err := writer.Write([]byte("APPDATA integration test\n")); err != nil {
		closer.Close()
		t.Fatalf("Write() failed: %v", err)
	}

	if err := closer.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	path := filepath.Join(base, appName, appName+".log")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	want := "APPDATA integration test\n"
	if string(content) != want {
		t.Fatalf(
			"log content = %q, want %q",
			string(content),
			want,
		)
	}
}
