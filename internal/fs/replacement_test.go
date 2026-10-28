package fs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestOpenFileRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()

	safeDir := filepath.Join(base, "safe")
	outsideDir := filepath.Join(base, "outside")

	mustMkdir(t, safeDir)
	mustMkdir(t, outsideDir)

	outsideFile := filepath.Join(outsideDir, "target.log")
	const original = "outside data\n"

	if err := os.WriteFile(outsideFile, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside the trusted directory
	// that resolves to a file outside the trusted root.
	link := filepath.Join(safeDir, "app.log")
	requireSymlink(t, outsideFile, link)

	root, err := os.OpenRoot(safeDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	writer, closer, err := openFile(root, "app.log", link)
	if err == nil {
		_ = closer.Close()
		t.Fatal("expected symlink escape to be rejected")
	}
	if writer != nil || closer != nil {
		t.Fatal("expected nil writer and closer on failure")
	}

	assertFileContent(t, outsideFile, original)
}

func TestOpenFileRejectsDirectorySymlinkEscape(t *testing.T) {
	base := t.TempDir()

	safeDir := filepath.Join(base, "safe")
	outsideDir := filepath.Join(base, "outside")

	mustMkdir(t, safeDir)
	mustMkdir(t, outsideDir)

	// Directory symlink pointing outside the trusted root.
	link := filepath.Join(safeDir, "redirect")
	requireSymlink(t, outsideDir, link)

	root, err := os.OpenRoot(safeDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	outsideFile := filepath.Join(outsideDir, "attack.log")

	writer, closer, err := openFile(
		root,
		filepath.Join("redirect", "attack.log"),
		outsideFile,
	)
	if err == nil {
		_ = closer.Close()
		t.Fatal("expected directory symlink escape to be rejected")
	}
	if writer != nil || closer != nil {
		t.Fatal("expected nil writer and closer on failure")
	}

	if _, err := os.Stat(outsideFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file unexpectedly created: %v", err)
	}
}

// TestOpenFileConcurrentDirectoryReplacement attempts to redirect
// log writes by repeatedly replacing a directory entry with
// symlinks targeting locations inside and outside the trusted root.
//
// The test verifies that no data reaches the outside target,
// regardless of whether individual open attempts succeed or fail.
func TestOpenFileConcurrentDirectoryReplacement(t *testing.T) {
	// Windows directory replacements have different sharing and
	// locking semantics. Test Windows junction replacement separately.
	if runtime.GOOS == "windows" {
		t.Skip("Unix-style directory replacement stress test")
	}

	base := t.TempDir()

	safeDir := filepath.Join(base, "safe")
	insideDir := filepath.Join(safeDir, "inside")
	outsideDir := filepath.Join(base, "outside")

	mustMkdir(t, insideDir)
	mustMkdir(t, outsideDir)

	outsideFile := filepath.Join(outsideDir, "attack.log")
	const original = "DO NOT MODIFY\n"

	if err := os.WriteFile(outsideFile, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(safeDir, "redirect")
	requireSymlink(t, insideDir, link)

	root, err := os.OpenRoot(safeDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	const iterations = 1000

	// A separate goroutine repeatedly changes the symlink target.
	// Removal and creation aren't atomic, so some opens may
	// legitimately fail while the entry is absent.
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			_ = os.Remove(link)

			target := outsideDir
			if i%2 == 0 {
				target = insideDir
			}

			_ = os.Symlink(target, link)
		}
	}()

	for i := 0; i < iterations; i++ {
		writer, closer, err := openFile(
			root,
			filepath.Join("redirect", "attack.log"),
			filepath.Join(link, "attack.log"),
		)

		if err != nil {
			// Expected when:
			// - the symlink is temporarily missing;
			// - it points outside the root;
			// - the filesystem changes during traversal.
			continue
		}

		if writer == nil || closer == nil {
			t.Fatal("successful open returned nil writer or closer")
		}

		if _, err := writer.Write([]byte("LOG ENTRY\n")); err != nil {
			_ = closer.Close()
			t.Fatalf("Write() failed: %v", err)
		}

		if err := closer.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}
	}

	wg.Wait()

	// The outside file must remain untouched.
	assertFileContent(t, outsideFile, original)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func requireSymlink(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		// Windows may require Developer Mode or symlink privileges.
		if runtime.GOOS == "windows" {
			t.Skipf("cannot create Windows symlink: %v", err)
		}
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != want {
		t.Fatalf(
			"file %q modified: got %q, want %q",
			path,
			data,
			want,
		)
	}
}
