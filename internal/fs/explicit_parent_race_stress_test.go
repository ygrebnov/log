//go:build !windows && stress

package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/ygrebnov/log/pkg/config"
)

// TestExplicitPathConcurrentParentReplacement continuously replaces
// a parent-directory symlink while OpenWriter initializes.
//
// This is a stress test, not a deterministic proof of the race
// between MkdirAll and OpenRoot.
func TestExplicitPathConcurrentParentReplacement(t *testing.T) {
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

	const iterations = 2000
	const marker = "race-test\n"

	var wg sync.WaitGroup
	swapErrors := make(chan error, 1)

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			target := trusted
			if i%2 == 0 {
				target = outside
			}

			if err := replaceParentSymlink(parent, target); err != nil {
				swapErrors <- err
				return
			}

			runtime.Gosched()
		}
	}()

	successes := 0
	failures := 0

	for i := 0; i < iterations; i++ {
		writer, closer, err := OpenWriter(
			&config.SinkConfig{Path: logPath},
			"test-app",
		)
		if err != nil {
			// The parent may change during initialization.
			failures++
			continue
		}

		if _, err := writer.Write([]byte(marker)); err != nil {
			_ = closer.Close()
			t.Fatal(err)
		}

		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}

		successes++
		runtime.Gosched()
	}

	wg.Wait()
	close(swapErrors)

	for err := range swapErrors {
		t.Fatalf("parent replacement failed: %v", err)
	}

	if successes == 0 {
		t.Fatal("no successful OpenWriter operations")
	}

	outsideFile := filepath.Join(outside, "app.log")

	data, err := os.ReadFile(outsideFile)
	switch {
	case os.IsNotExist(err):
		t.Log("no redirected writes observed")
	case err != nil:
		t.Fatal(err)
	default:
		outsideWrites := strings.Count(string(data), marker)
		t.Logf("redirected writes: %d", outsideWrites)
	}

	t.Logf(
		"OpenWriter successes: %d, failures: %d",
		successes,
		failures,
	)
}
