//go:build !windows

package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ygrebnov/log/pkg/config"
)

// TestExplicitPathRedirectedParent demonstrates that an explicit
// log path can resolve to a different directory if an attacker
// replaces one of its parent path components.
//
// This test is deterministic. It does not claim to reproduce
// the exact MkdirAll/OpenRoot race window.
func TestExplicitPathRedirectedParent(t *testing.T) {
	base := t.TempDir()

	trusted := filepath.Join(base, "trusted")
	outside := filepath.Join(base, "outside")
	parent := filepath.Join(base, "current")

	for _, dir := range []string{trusted, outside} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Symlink(trusted, parent); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(parent, "app.log")

	// Verify that the configured parent initially points
	// to the trusted destination.
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate an attacker replacing the directory entry.
	if err := replaceParentSymlink(parent, outside); err != nil {
		t.Fatal(err)
	}

	writer, closer, err := OpenWriter(
		&config.SinkConfig{Path: logPath},
		"test-app",
	)
	if err != nil {
		t.Fatalf("OpenWriter: %v", err)
	}

	const marker = "redirected write\n"

	if _, err := writer.Write([]byte(marker)); err != nil {
		_ = closer.Close()
		t.Fatal(err)
	}

	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}

	// The file was actually created in the redirected directory.
	outsideFile := filepath.Join(outside, "app.log")

	content, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("outside file not found: %v", err)
	}

	if string(content) != marker {
		t.Fatalf("unexpected outside content: %q", content)
	}

	t.Log("confirmed: explicit path followed replaced parent")
}

// replaceParentSymlink atomically replaces a symlink with another
// symlink targeting a different directory.
//
// On Unix, renaming a symlink over an existing symlink replaces
// the directory entry atomically.
func replaceParentSymlink(link, target string) error {
	next := link + ".next"

	if err := os.Symlink(target, next); err != nil {
		return err
	}

	if err := os.Rename(next, link); err != nil {
		_ = os.Remove(next)
		return err
	}

	return nil
}
