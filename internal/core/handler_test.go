package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/config"
	"github.com/ygrebnov/log/pkg/types"
)

func newHandlerTestSink(level types.Level, queueSize int) *Sink {
	return &Sink{
		records: make(chan types.Record, queueSize),
		level:   level,
	}
}

func newHandlerTestRecord(message string, fields ...types.Field) types.Record {
	return types.NewRecord(
		time.Now(),
		types.LevelInfo,
		message,
		fields...,
	)
}

func TestHandler_NewHandler(t *testing.T) {
	cfg := &config.SinkConfig{
		Kind:          types.KindStdOut,
		Format:        types.FormatText,
		Level:         types.LevelInfo,
		QueueSize:     16,
		BufferSize:    4096,
		FlushInterval: time.Second,
	}

	sink, err := NewSink(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("NewSink() error: %v", err)
	}

	sinks := []*Sink{sink}

	handler := NewHandler(sinks...)

	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}

	if handler.state == nil {
		t.Fatal("handler state is nil")
	}

	if len(handler.state.sinks) != 1 {
		t.Fatalf(
			"sink count = %d, want 1",
			len(handler.state.sinks),
		)
	}

	if handler.state.sinks[0] != sink {
		t.Fatal("unexpected sink")
	}

	// NewHandler must take ownership of the sink slice.
	sinks[0] = nil

	if handler.state.sinks[0] != sink {
		t.Fatal("handler sink changed after caller modified its slice")
	}

	// The sink is started by NewHandler.
	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestHandler_LogRecord_LevelFiltering(t *testing.T) {
	infoSink := newHandlerTestSink(types.LevelInfo, 2)
	errorSink := newHandlerTestSink(types.LevelError, 1)

	// Construct the handler directly to inspect the queues without
	// starting sink workers.
	handler := &Handler{
		state: &handlerState{
			sinks: []*Sink{infoSink, errorSink},
		},
	}

	record := newHandlerTestRecord("info record")

	handler.LogRecord(record, false)

	if got := len(infoSink.records); got != 1 {
		t.Errorf("info sink queue length = %d, want 1", got)
	}

	if got := len(errorSink.records); got != 0 {
		t.Errorf("error sink queue length = %d, want 0", got)
	}

	record.Level = types.LevelError
	record.Message = "error record"

	handler.LogRecord(record, false)

	if got := len(infoSink.records); got != 2 {
		t.Errorf("info sink queue length = %d, want 2", got)
	}

	if got := len(errorSink.records); got != 1 {
		t.Errorf("error sink queue length = %d, want 1", got)
	}
}

func TestHandler_LogRecord_CopyFields(t *testing.T) {
	tests := []struct {
		name       string
		copyFields bool
		withFields bool
		wantCopy   bool
	}{
		{
			name:       "no copy requested",
			copyFields: false,
			withFields: false,
			wantCopy:   false,
		},
		{
			name:       "explicit copy",
			copyFields: true,
			withFields: false,
			wantCopy:   true,
		},
		{
			name:       "bound fields require copy",
			copyFields: false,
			withFields: true,
			wantCopy:   true,
		},
		{
			name:       "explicit copy with bound fields",
			copyFields: true,
			withFields: true,
			wantCopy:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := newHandlerTestSink(types.LevelInfo, 1)

			handler := &Handler{
				state: &handlerState{
					sinks: []*Sink{sink},
				},
			}

			if tt.withFields {
				handler = handler.With(
					types.NewStringField(
						keys.New("service"),
						"api",
					),
				)
			}

			fields := []types.Field{
				types.NewStringField(
					keys.New("request_id"),
					"123",
				),
			}

			record := newHandlerTestRecord("test", fields...)

			handler.LogRecord(record, tt.copyFields)

			var queued types.Record

			select {
			case queued = <-sink.records:
			default:
				t.Fatal("record was not enqueued")
			}

			// Mutating the original field slice must only affect
			// the queued record when copying was not requested.
			record.Fields[0] = types.NewStringField(
				keys.New("request_id"),
				"modified",
			)

			if tt.wantCopy {
				for _, field := range queued.Fields {
					if field.String() == "modified" {
						t.Fatal("queued record shares caller-owned field slice")
					}
				}
			} else {
				if queued.Fields[0].String() != "modified" {
					t.Fatal("expected queued record to share field slice")
				}
			}

			if tt.withFields {
				if got := queued.Fields[0].Key(); got != "service" {
					t.Errorf("first field key = %q, want service", got)
				}

				if got := queued.Fields[1].Key(); got != "request_id" {
					t.Errorf("second field key = %q, want request_id", got)
				}
			}
		})
	}
}

func TestHandler_With(t *testing.T) {
	sink := newHandlerTestSink(types.LevelInfo, 10)

	base := &Handler{
		state: &handlerState{
			sinks: []*Sink{sink},
		},
	}

	first := base.With(
		types.NewStringField(keys.New("service"), "api"),
	)

	second := first.With(
		types.NewStringField(keys.New("request_id"), "123"),
	)

	if first.state != base.state {
		t.Fatal("first derived handler does not share state")
	}

	if second.state != base.state {
		t.Fatal("second derived handler does not share state")
	}

	if len(base.fields) != 0 {
		t.Fatalf("base field count = %d, want 0", len(base.fields))
	}

	if len(first.fields) != 1 {
		t.Fatalf("first field count = %d, want 1", len(first.fields))
	}

	if len(second.fields) != 2 {
		t.Fatalf("second field count = %d, want 2", len(second.fields))
	}

	// Ensure derived handlers do not share mutable field slices.
	second.fields[0] = types.NewStringField(
		keys.New("service"),
		"modified",
	)

	if got := first.fields[0].String(); got != "api" {
		t.Errorf("first handler field = %q, want api", got)
	}
}

func TestHandler_With_AfterClose(t *testing.T) {
	cfg := &config.SinkConfig{
		Kind:          types.KindStdOut,
		Format:        types.FormatText,
		Level:         types.LevelInfo,
		QueueSize:     16,
		BufferSize:    4096,
		FlushInterval: time.Second,
	}

	sink, err := NewSink(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("NewSink() error: %v", err)
	}

	handler := NewHandler(sink)

	if err := handler.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	derived := handler.With(
		types.NewStringField(keys.New("service"), "api"),
	)

	if derived == handler {
		t.Fatal("With() returned the original handler")
	}

	if derived.state != handler.state {
		t.Fatal("derived handler does not share state")
	}

	// Logging after shutdown must be ignored.
	derived.LogRecord(newHandlerTestRecord("after close"), true)
}

func TestHandler_Close_Idempotent(t *testing.T) {
	cfg := &config.SinkConfig{
		Kind:          types.KindStdOut,
		Format:        types.FormatText,
		Level:         types.LevelInfo,
		QueueSize:     16,
		BufferSize:    4096,
		FlushInterval: time.Second,
	}

	sink, err := NewSink(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("NewSink() error: %v", err)
	}

	handler := NewHandler(sink)

	if err := handler.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}

	if err := handler.Close(); err != nil {
		t.Fatalf("second Close() error: %v", err)
	}
}

func TestHandler_Close_SharedLifecycle(t *testing.T) {
	cfg := &config.SinkConfig{
		Kind:          types.KindStdOut,
		Format:        types.FormatText,
		Level:         types.LevelInfo,
		QueueSize:     16,
		BufferSize:    4096,
		FlushInterval: time.Second,
	}

	sink, err := NewSink(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("NewSink() error: %v", err)
	}

	base := NewHandler(sink)

	derived := base.With(
		types.NewStringField(keys.New("service"), "api"),
	)

	if derived.state != base.state {
		t.Fatal("derived handler does not share state")
	}

	if err := derived.Close(); err != nil {
		t.Fatalf("derived Close() error: %v", err)
	}

	if !base.state.closed {
		t.Fatal("shared state is not closed")
	}

	// Closing either handler again must remain safe.
	if err := base.Close(); err != nil {
		t.Fatalf("base Close() error: %v", err)
	}

	base.LogRecord(newHandlerTestRecord("ignored"), false)
	derived.LogRecord(newHandlerTestRecord("also ignored"), false)
}

func TestHandler_ConcurrentLogRecordAndClose(t *testing.T) {
	cfg := &config.SinkConfig{
		Kind:          types.KindStdOut,
		Format:        types.FormatText,
		Level:         types.LevelInfo,
		QueueSize:     16,
		BufferSize:    4096,
		FlushInterval: time.Second,
	}

	sink, err := NewSink(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("NewSink() error: %v", err)
	}

	handler := NewHandler(sink)

	const (
		goroutines = 16
		records    = 100
	)

	var wg sync.WaitGroup

	start := make(chan struct{})

	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()

			<-start

			for i := 0; i < records; i++ {
				handler.LogRecord(
					newHandlerTestRecord("concurrent"),
					false,
				)
			}
		}()
	}

	close(start)

	closeDone := make(chan error, 1)

	go func() {
		closeDone <- handler.Close()
	}()

	wg.Wait()

	if err := <-closeDone; err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Must remain safe after shutdown.
	handler.LogRecord(newHandlerTestRecord("after close"), false)
}

func TestHandler_ConcurrentWithAndClose(t *testing.T) {
	cfg := &config.SinkConfig{
		Kind:          types.KindStdOut,
		Format:        types.FormatText,
		Level:         types.LevelInfo,
		QueueSize:     16,
		BufferSize:    4096,
		FlushInterval: time.Second,
	}

	sink, err := NewSink(context.Background(), cfg, "")
	if err != nil {
		t.Fatalf("NewSink() error: %v", err)
	}

	handler := NewHandler(sink)

	const (
		goroutines = 16
		iterations = 100
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	start := make(chan struct{})

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()

			<-start

			for i := 0; i < iterations; i++ {
				derived := handler.With(
					types.NewStringField(
						keys.New("service"),
						"api",
					),
				)

				derived.LogRecord(
					newHandlerTestRecord("test"),
					false,
				)
			}
		}()
	}

	close(start)

	closeDone := make(chan error, 1)

	go func() {
		closeDone <- handler.Close()
	}()

	wg.Wait()

	if err := <-closeDone; err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}
