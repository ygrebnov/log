package tests

import (
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

type testRecord struct {
	level   log.Level
	message string
}

func newStdoutLogger(t *testing.T, level log.Level) *log.Logger {
	t.Helper()

	cfg := &config.Config{
		Sinks: []config.SinkConfig{
			textSink(types.KindStdOut, level),
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

	return logger
}

func textSink(kind types.Kind, level types.Level) config.SinkConfig {
	return config.SinkConfig{
		Kind:   kind,
		Format: types.FormatText,
		Level:  level,
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	return captureFileOutput(t, &os.Stdout, fn)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	return captureFileOutput(t, &os.Stderr, fn)
}

func captureFileOutput(
	t *testing.T,
	target **os.File,
	fn func(),
) string {
	t.Helper()

	original := *target

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}

	*target = w

	var (
		wg      sync.WaitGroup
		output  []byte
		readErr error
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		output, readErr = io.ReadAll(r)
	}()

	defer func() {
		*target = original

		_ = w.Close()
		wg.Wait()
		_ = r.Close()

		if readErr != nil {
			t.Errorf("reading captured output: %v", readErr)
		}
	}()

	fn()

	*target = original

	if err := w.Close(); err != nil {
		t.Fatalf("closing captured writer: %v", err)
	}

	wg.Wait()

	if readErr != nil {
		t.Fatalf("reading captured output: %v", readErr)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("closing captured reader: %v", err)
	}

	return string(output)
}

func assertContainsAll(t *testing.T, output string, values []string) {
	t.Helper()

	for _, value := range values {
		if !strings.Contains(output, value) {
			t.Errorf("output does not contain %q:\n%s", value, output)
		}
	}
}

func assertContainsNone(t *testing.T, output string, values []string) {
	t.Helper()

	for _, value := range values {
		if strings.Contains(output, value) {
			t.Errorf("output unexpectedly contains %q:\n%s", value, output)
		}
	}
}

func countNonEmptyLines(output string) int {
	count := 0

	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	return count
}

func countMessage(output, message string) int {
	count := 0

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, message) {
			count++
		}
	}

	return count
}
