package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ygrebnov/log"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

func TestLogger_ThreadSafety(t *testing.T) {
	t.Run("concurrent Log", func(t *testing.T) {
		const (
			goroutines = 32
			records    = 100
		)

		output := captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelInfo)

			var wg sync.WaitGroup
			wg.Add(goroutines)

			for g := 0; g < goroutines; g++ {
				go func(g int) {
					defer wg.Done()

					for i := 0; i < records; i++ {
						logger.Log(
							log.LevelInfo,
							fmt.Sprintf("concurrent-%d-%d", g, i),
						)
					}
				}(g)
			}

			wg.Wait()

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})

		got := countNonEmptyLines(output)
		want := goroutines * records

		if got != want {
			t.Fatalf("line count = %d, want %d", got, want)
		}
	})

	t.Run("concurrent Close", func(t *testing.T) {
		const closers = 32

		output := captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelInfo)

			logger.Log(log.LevelInfo, "before concurrent close")

			var wg sync.WaitGroup
			errs := make(chan error, closers)

			wg.Add(closers)

			for i := 0; i < closers; i++ {
				go func() {
					defer wg.Done()
					errs <- logger.Close()
				}()
			}

			wg.Wait()
			close(errs)

			for err := range errs {
				if err != nil {
					t.Errorf("Close() error: %v", err)
				}
			}
		})

		if got := countMessage(output, "info before concurrent close"); got != 1 {
			t.Fatalf("record count = %d, want 1:\n%s", got, output)
		}
	})

	t.Run("concurrent Log and Close", func(t *testing.T) {
		const (
			goroutines = 16
			records    = 100
		)

		_ = captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelInfo)

			start := make(chan struct{})

			var logWG sync.WaitGroup
			logWG.Add(goroutines)

			for g := 0; g < goroutines; g++ {
				go func(g int) {
					defer logWG.Done()

					<-start

					for i := 0; i < records; i++ {
						logger.Log(
							log.LevelInfo,
							fmt.Sprintf("race-%d-%d", g, i),
						)
					}
				}(g)
			}

			close(start)

			var closeWG sync.WaitGroup
			closeWG.Add(1)

			var closeErr error

			go func() {
				defer closeWG.Done()
				closeErr = logger.Close()
			}()

			logWG.Wait()
			closeWG.Wait()

			if closeErr != nil {
				t.Fatalf("Close() error: %v", closeErr)
			}

			// Must remain safe after Close has won the race.
			logger.Log(log.LevelInfo, "after close")
		})
	})

	t.Run("concurrent Log to stdout and file", func(t *testing.T) {
		const (
			goroutines = 16
			records    = 50
		)

		path := filepath.Join(t.TempDir(), "concurrent.log")

		cfg := &config.Config{
			AppName: "log-e2e",
			Sinks: []config.SinkConfig{
				{
					Kind:   types.KindStdOut,
					Format: types.FormatText,
					Level:  types.LevelInfo,
				},
				{
					Kind:   types.KindFile,
					Format: types.FormatText,
					Level:  types.LevelInfo,
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

			var wg sync.WaitGroup
			wg.Add(goroutines)

			for g := 0; g < goroutines; g++ {
				go func(g int) {
					defer wg.Done()

					for i := 0; i < records; i++ {
						logger.Log(
							log.LevelInfo,
							fmt.Sprintf("multi-%d-%d", g, i),
						)
					}
				}(g)
			}

			wg.Wait()

			if err := logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})

		fileData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}

		want := goroutines * records

		if got := countNonEmptyLines(stdout); got != want {
			t.Errorf("stdout line count = %d, want %d", got, want)
		}

		if got := countNonEmptyLines(string(fileData)); got != want {
			t.Errorf("file line count = %d, want %d", got, want)
		}
	})
}
