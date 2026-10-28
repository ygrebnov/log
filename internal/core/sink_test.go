package core

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ygrebnov/log/pkg/types"
)

type sinkTestFormatter struct {
	format func(types.Record) ([]byte, error)
}

func (f sinkTestFormatter) Format(r types.Record) ([]byte, error) {
	return f.format(r)
}

type sinkTestWriter struct {
	write func([]byte) (int, error)
}

func (w sinkTestWriter) Write(p []byte) (int, error) {
	return w.write(p)
}

type sinkTestCloser struct {
	close func() error
}

func (c sinkTestCloser) Close() error {
	return c.close()
}

func newSinkTestRecord(message string) types.Record {
	return types.NewRecord(
		time.Now(),
		types.LevelInfo,
		message,
	)
}

func newSinkTestInstance(
	out io.Writer,
	f formatter,
	queueSize int,
) *Sink {
	return &Sink{
		records: make(chan types.Record, queueSize),
		done:    make(chan struct{}),

		level:     types.LevelInfo,
		formatter: f,

		writer: bufio.NewWriterSize(out, 64),

		flushInterval: time.Hour,
	}
}

func TestSink_Enqueue_QueueFull(t *testing.T) {
	sink := newSinkTestInstance(
		io.Discard,
		sinkTestFormatter{
			format: func(types.Record) ([]byte, error) {
				return []byte("record\n"), nil
			},
		},
		1,
	)

	first := newSinkTestRecord("first")
	second := newSinkTestRecord("second")

	sink.enqueue(first)

	done := make(chan struct{})

	go func() {
		sink.enqueue(second)
		close(done)
	}()

	select {
	case <-done:
		// Enqueue must not block when the queue is full.

	case <-time.After(time.Second):
		t.Fatal("enqueue blocked on a full queue")
	}

	if got := len(sink.records); got != 1 {
		t.Fatalf("queue length = %d, want 1", got)
	}

	record := <-sink.records

	if record.Message != "first" {
		t.Errorf(
			"queued message = %q, want %q",
			record.Message,
			"first",
		)
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestSink_Write_FormatterError(t *testing.T) {
	errFormat := errors.New("format failed")

	var output bytes.Buffer

	var calls int

	sink := newSinkTestInstance(
		&output,
		sinkTestFormatter{
			format: func(r types.Record) ([]byte, error) {
				calls++

				if r.Message == "invalid" {
					return nil, errFormat
				}

				return []byte(r.Message + "\n"), nil
			},
		},
		4,
	)

	sink.write(newSinkTestRecord("first"))
	sink.write(newSinkTestRecord("invalid"))
	sink.write(newSinkTestRecord("second"))

	if err := sink.getError(); err != nil {
		t.Fatalf("unexpected sink error: %v", err)
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if got, want := output.String(), "first\nsecond\n"; got != want {
		t.Errorf(
			"output = %q, want %q",
			got,
			want,
		)
	}

	if calls != 3 {
		t.Errorf("formatter calls = %d, want 3", calls)
	}
}

func TestSink_Write_WriterError(t *testing.T) {
	errWrite := errors.New("write failed")

	var writeCalls atomic.Int32
	var formatCalls atomic.Int32

	writer := sinkTestWriter{
		write: func([]byte) (int, error) {
			writeCalls.Add(1)
			return 0, errWrite
		},
	}

	sink := newSinkTestInstance(
		writer,
		sinkTestFormatter{
			format: func(types.Record) ([]byte, error) {
				formatCalls.Add(1)

				return bytes.Repeat([]byte("x"), 1024), nil
			},
		},
		4,
	)

	sink.write(newSinkTestRecord("first"))

	firstErr := sink.getError()

	if firstErr == nil {
		t.Fatal("expected writer error")
	}

	// A permanently failed sink must not attempt subsequent writes.
	sink.write(newSinkTestRecord("second"))

	if got := formatCalls.Load(); got != 1 {
		t.Errorf("formatter calls = %d, want 1", got)
	}

	if got := writeCalls.Load(); got != 1 {
		t.Errorf("writer calls = %d, want 1", got)
	}

	if err := sink.Close(); err != firstErr {
		t.Errorf(
			"Close() error = %v, want original error %v",
			err,
			firstErr,
		)
	}
}

func TestSink_Flush_Error(t *testing.T) {
	errFlush := errors.New("flush failed")

	var writeCalls atomic.Int32
	var formatCalls atomic.Int32

	writer := sinkTestWriter{
		write: func([]byte) (int, error) {
			writeCalls.Add(1)
			return 0, errFlush
		},
	}

	sink := newSinkTestInstance(
		writer,
		sinkTestFormatter{
			format: func(r types.Record) ([]byte, error) {
				formatCalls.Add(1)
				return []byte(r.Message + "\n"), nil
			},
		},
		4,
	)

	sink.write(newSinkTestRecord("first"))

	if err := sink.getError(); err != nil {
		t.Fatalf("unexpected error before flush: %v", err)
	}

	sink.flush()

	firstErr := sink.getError()

	if firstErr == nil {
		t.Fatal("expected flush error")
	}

	sink.write(newSinkTestRecord("second"))

	if got := formatCalls.Load(); got != 1 {
		t.Errorf("formatter calls = %d, want 1", got)
	}

	if got := writeCalls.Load(); got != 1 {
		t.Errorf("writer calls = %d, want 1", got)
	}

	if err := sink.Close(); err != firstErr {
		t.Errorf(
			"Close() error = %v, want original error %v",
			err,
			firstErr,
		)
	}
}

func TestSink_Close_OutputError(t *testing.T) {
	errClose := errors.New("close failed")

	var closeCalls atomic.Int32

	sink := newSinkTestInstance(
		io.Discard,
		sinkTestFormatter{
			format: func(types.Record) ([]byte, error) {
				return []byte("record\n"), nil
			},
		},
		4,
	)

	sink.closer = sinkTestCloser{
		close: func() error {
			closeCalls.Add(1)
			return errClose
		},
	}

	if err := sink.Close(); !errors.Is(err, errClose) {
		t.Fatalf(
			"Close() error = %v, want %v",
			err,
			errClose,
		)
	}

	if err := sink.Close(); !errors.Is(err, errClose) {
		t.Fatalf(
			"second Close() error = %v, want %v",
			err,
			errClose,
		)
	}

	if got := closeCalls.Load(); got != 1 {
		t.Errorf("close calls = %d, want 1", got)
	}
}

func TestSink_ConcurrentEnqueueAndClose(t *testing.T) {
	sink := newSinkTestInstance(
		io.Discard,
		sinkTestFormatter{
			format: func(types.Record) ([]byte, error) {
				return []byte("record\n"), nil
			},
		},
		64,
	)

	const (
		goroutines = 32
		records    = 1000
	)

	start := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()

			<-start

			for i := 0; i < records; i++ {
				sink.enqueue(newSinkTestRecord("concurrent"))
			}
		}()
	}

	close(start)

	closeDone := make(chan error, 1)

	go func() {
		closeDone <- sink.Close()
	}()

	wg.Wait()

	if err := <-closeDone; err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Enqueue after shutdown must also remain safe.
	sink.enqueue(newSinkTestRecord("after close"))
}
