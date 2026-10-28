package fs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOpenFileRejectsJunctionEscape(t *testing.T) {
	base := t.TempDir()

	safeDir := filepath.Join(base, "safe")
	outsideDir := filepath.Join(base, "outside")

	if err := os.MkdirAll(safeDir, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(outsideDir, 0o700); err != nil {
		t.Fatal(err)
	}

	junction := filepath.Join(safeDir, "redirect")

	cmd := exec.Command(
		"cmd", "/c", "mklink", "/J", junction, outsideDir,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("cannot create junction: %v: %s", err, output)
	}

	root, err := os.OpenRoot(safeDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	outsideFile := filepath.Join(outsideDir, "attack.log")

	writer, closer, err := openFile(
		root,
		filepath.Join("redirect", "attack.log"),
		filepath.Join(junction, "attack.log"),
	)

	if err == nil {
		_ = closer.Close()
		t.Fatal("expected junction escape to be rejected")
	}

	if writer != nil || closer != nil {
		t.Fatal("expected nil writer and closer")
	}

	if _, err := os.Stat(outsideFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file unexpectedly created: %v", err)
	}
}
