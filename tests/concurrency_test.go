package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ygrebnov/keys"
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
			queueSize := goroutines * records
			logger := newStdoutLogger(t, log.LevelInfo, &queueSize)

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
			logger := newStdoutLogger(t, log.LevelInfo, nil)

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

		if got := countMessage(output, "INFO before concurrent close"); got != 1 {
			t.Fatalf("record count = %d, want 1:\n%s", got, output)
		}
	})

	t.Run("concurrent Log and Close", func(t *testing.T) {
		const (
			goroutines = 16
			records    = 100
		)

		_ = captureStdout(t, func() {
			logger := newStdoutLogger(t, log.LevelInfo, nil)

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
		queueSize := goroutines * records

		cfg := &config.Config{
			AppName: "log-e2e",
			Sinks: []config.SinkConfig{
				{
					Kind:      types.KindStdOut,
					Format:    types.FormatText,
					Level:     types.LevelInfo,
					QueueSize: queueSize,
				},
				{
					Kind:      types.KindFile,
					Format:    types.FormatText,
					Level:     types.LevelInfo,
					Path:      path,
					QueueSize: queueSize,
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

			if errClose := logger.Close(); errClose != nil {
				t.Fatalf("Close() error: %v", errClose)
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

func TestLogger_LogRecord_SnapshotsRecord(t *testing.T) {
	const (
		fieldCount    = 10_000
		originalValue = "original"
		mutatedValue  = "mutated"
	)

	path := filepath.Join(t.TempDir(), "log-record.log")

	cfg := log.Config{
		Sinks: []log.SinkConfig{
			{
				Kind: log.KindFile,
				Path: path,
			},
		},
	}

	if err := cfg.ApplyDefaults(); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	logger, err := log.NewLogger(&cfg)
	if err != nil {
		t.Fatalf("NewLogger() error: %v", err)
	}

	fields := make([]log.Field, fieldCount)
	for i := range fields {
		fields[i] = log.String(
			keys.New(fmt.Sprintf("field_%d", i)),
			originalValue,
		)
	}

	record := log.NewRecord(
		time.Now(),
		log.LevelInfo,
		"record snapshot",
		fields...,
	)

	logger.LogRecord(record)

	// LogRecord must take ownership of a snapshot before returning.
	//
	// Mutate the Record concurrently while the sink may still be processing
	// the queued record. Without snapshotting in LogRecord, this should be
	// detected by `go test -race`.
	stop := make(chan struct{})
	done := make(chan struct{})

	originalFields := make([]log.Field, fieldCount)
	mutatedFields := make([]log.Field, fieldCount)

	for i := range originalFields {
		key := keys.New(fmt.Sprintf("field_%d", i))

		originalFields[i] = log.String(key, originalValue)
		mutatedFields[i] = log.String(key, mutatedValue)
	}

	go func() {
		defer close(done)

		for {
			select {
			case <-stop:
				return
			default:
			}

			for i := range record.Fields {
				record.Fields[i] = mutatedFields[i]
			}

			for i := range record.Fields {
				record.Fields[i] = originalFields[i]
			}
		}
	}()

	if errClose := logger.Close(); errClose != nil {
		close(stop)
		<-done
		t.Fatalf("Close() error: %v", errClose)
	}

	close(stop)
	<-done

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	output := string(data)

	if !strings.Contains(output, `field_0="original"`) {
		t.Errorf(
			"output does not contain original field value:\n%s",
			output,
		)
	}

	if strings.Contains(output, `="mutated"`) {
		t.Errorf(
			"output contains value written after LogRecord returned:\n%s",
			output,
		)
	}
}

func TestLogger_ConcurrentWithAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "with-close.log")

	logger, err := log.NewFileLogger(log.WithPath(path))
	if err != nil {
		t.Fatalf("NewFileLogger() error: %v", err)
	}

	const (
		goroutines = 16
		iterations = 100
	)

	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()

			<-start

			for i := 0; i < iterations; i++ {
				derived := logger.With(
					log.String(keys.New("worker"), fmt.Sprintf("%d", g)),
				)

				derived.Info("concurrent With and Close")
			}
		}(g)
	}

	close(start)

	closeDone := make(chan error, 1)

	go func() {
		closeDone <- logger.Close()
	}()

	wg.Wait()

	if err := <-closeDone; err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Creating a derived logger after shutdown must remain safe.
	derived := logger.With(
		log.String(keys.New("service"), "after-close"),
	)

	derived.Info("ignored after close")
	logger.Info("also ignored after close")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	output := string(data)

	if strings.Contains(output, "ignored after close") {
		t.Errorf("records written after Close():\n%s", output)
	}
}

func TestLogger_ConcurrentParentChildLogging(t *testing.T) {
	const (
		goroutines = 8
		records    = 100
	)

	path := filepath.Join(t.TempDir(), "parent-child.log")
	queueSize := goroutines * records * 3

	cfg := &log.Config{
		Sinks: []log.SinkConfig{
			{
				Kind:      log.KindFile,
				Path:      path,
				QueueSize: queueSize,
			},
		},
	}

	if err := cfg.ApplyDefaults(); err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	base, err := log.NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger() error: %v", err)
	}

	first := base.With(
		log.String(keys.New("component"), "first"),
	)

	second := base.With(
		log.String(keys.New("component"), "second"),
	)

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()

			for i := 0; i < records; i++ {
				base.Info(
					fmt.Sprintf("base-%d-%d", g, i),
				)
			}
		}(g)

		go func(g int) {
			defer wg.Done()

			for i := 0; i < records; i++ {
				first.Info(
					fmt.Sprintf("first-%d-%d", g, i),
				)
			}
		}(g)

		go func(g int) {
			defer wg.Done()

			for i := 0; i < records; i++ {
				second.Info(
					fmt.Sprintf("second-%d-%d", g, i),
				)
			}
		}(g)
	}

	wg.Wait()

	if err := base.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if got := len(lines); got != queueSize {
		t.Fatalf("line count = %d, want %d", got, queueSize)
	}

	counts := map[string]int{
		"base":   0,
		"first":  0,
		"second": 0,
	}

	for _, line := range lines {
		switch {
		case strings.Contains(line, "INFO base-"):
			if strings.Contains(line, `component=`) {
				t.Errorf("base record contains derived fields: %s", line)
			}

			counts["base"]++

		case strings.Contains(line, "INFO first-"):
			if !strings.Contains(line, `component="first"`) {
				t.Errorf("first record has incorrect fields: %s", line)
			}

			counts["first"]++

		case strings.Contains(line, "INFO second-"):
			if !strings.Contains(line, `component="second"`) {
				t.Errorf("second record has incorrect fields: %s", line)
			}

			counts["second"]++

		default:
			t.Errorf("unexpected record: %s", line)
		}
	}

	want := goroutines * records

	for name, got := range counts {
		if got != want {
			t.Errorf("%s records = %d, want %d", name, got, want)
		}
	}
}

func TestLogger_ConcurrentChildAndParentClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parent-child-close.log")

	parent, err := log.NewFileLogger(log.WithPath(path))
	if err != nil {
		t.Fatalf("NewFileLogger() error: %v", err)
	}

	child := parent.With(
		log.String(keys.New("component"), "child"),
	)

	child.Info("before close")

	const closers = 16

	var wg sync.WaitGroup
	errs := make(chan error, closers)

	wg.Add(closers)

	for i := 0; i < closers; i++ {
		go func(i int) {
			defer wg.Done()

			if i%2 == 0 {
				errs <- parent.Close()
			} else {
				errs <- child.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	output := string(data)

	if got := countNonEmptyLines(output); got != 1 {
		t.Fatalf("line count = %d, want 1:\n%s", got, output)
	}

	if !strings.Contains(
		output,
		`INFO before close, component="child"`,
	) {
		t.Errorf("expected record not found:\n%s", output)
	}
}
