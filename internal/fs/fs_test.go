package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ygrebnov/log/pkg/config"
)

func TestValidateBaseWithinHome(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "alice")
	if runtime.GOOS == "windows" {
		home = `C:\Users\alice`
	}

	tests := []struct {
		name    string
		base    string
		wantErr bool
	}{
		{"home itself", home, false},
		{"child", filepath.Join(home, "data"), false},
		{"nested child", filepath.Join(home, "data", "logs"), false},
		{"sibling", filepath.Join(filepath.Dir(home), "bob"), true},
		{"prefix sibling", home + "2", true},
		{"parent", filepath.Dir(home), true},
		{"relative", "relative/path", true},
		{"empty", "", true},
		{"traversal outside", filepath.Join(home, "..", "other"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseWithinHome(home, tt.base)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"validateBaseWithinHome(%q, %q) error = %v; wantErr %v",
					home, tt.base, err, tt.wantErr,
				)
			}
		})
	}
}

func TestOpenWriterExplicitPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "app.log")

	w, closer, err := OpenWriter(&config.SinkConfig{Path: path}, "")
	if err != nil {
		t.Fatalf("OpenWriter() error = %v", err)
	}

	if _, err := w.Write([]byte("hello\n")); err != nil {
		_ = closer.Close()
		t.Fatal(err)
	}
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("contents = %q, want %q", got, "hello\n")
	}
}

func TestOpenWriterExistingFileAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	w, closer, err := OpenWriter(&config.SinkConfig{Path: path}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("second\n")); err != nil {
		_ = closer.Close()
		t.Fatal(err)
	}
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\nsecond\n" {
		t.Fatalf("contents = %q", got)
	}
}

func TestOpenWriterRejectsDirectoryDestination(t *testing.T) {
	_, closer, err := OpenWriter(
		&config.SinkConfig{Path: t.TempDir()},
		"",
	)
	if closer != nil {
		_ = closer.Close()
	}
	if err == nil {
		t.Fatal("expected error for directory destination")
	}
}

func TestOpenWriterRejectsSymlinkOutsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require privileges on Windows")
	}

	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.log")
	if err := os.WriteFile(outside, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "linked.log")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, closer, err := OpenWriter(&config.SinkConfig{Path: link}, "")
	if closer != nil {
		_ = closer.Close()
	}
	if err == nil {
		t.Fatal("expected error for symlink escaping root")
	}

	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unchanged" {
		t.Fatalf("outside file modified: %q", got)
	}
}

func TestOpenWriterDefaultDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows home-directory resolution differs")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	if runtime.GOOS != "darwin" {
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	}

	appName := "logger-test-app"
	w, closer, err := OpenWriter(&config.SinkConfig{}, appName)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Write([]byte("default\n")); err != nil {
		_ = closer.Close()
		t.Fatal(err)
	}
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	dir, err := getLogDir(appName)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, appName+".log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "default\n" {
		t.Fatalf("contents = %q", got)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("directory permissions = %04o; want private", info.Mode().Perm())
	}
}

func TestOpenWriterRejectsInsecureExistingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACL validation is handled separately")
	}

	path := filepath.Join(t.TempDir(), "app.log")

	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, closer, err := OpenWriter(&config.SinkConfig{Path: path}, "")
	if closer != nil {
		_ = closer.Close()
	}

	if err == nil {
		t.Fatal("expected error for insecure file permissions")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o644 {
		t.Fatalf("permissions changed to %04o", info.Mode().Perm())
	}
}

func TestOpenWriterPreservesExistingPrivateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACL validation is handled separately")
	}

	path := filepath.Join(t.TempDir(), "app.log")

	if err := os.WriteFile(path, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	w, closer, err := OpenWriter(&config.SinkConfig{Path: path}, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Write([]byte("appended\n")); err != nil {
		_ = closer.Close()
		t.Fatal(err)
	}

	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions changed to %04o", info.Mode().Perm())
	}
}

func TestOpenWriterRejectsInsecureDefaultDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACL validation is handled separately")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	if runtime.GOOS != "darwin" {
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	}

	appName := "logger-test-app"
	logDir, err := getLogDir(appName)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, closer, err := OpenWriter(&config.SinkConfig{}, appName)
	if closer != nil {
		_ = closer.Close()
	}

	if err == nil {
		t.Fatal("expected error for insecure application directory permissions")
	}

	info, err := os.Stat(logDir)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o755 {
		t.Fatalf("directory permissions changed to %04o", info.Mode().Perm())
	}
}
