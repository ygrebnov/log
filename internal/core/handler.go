package core

import (
	"errors"
	"sync"

	"github.com/ygrebnov/log/pkg/types"
)

type Handler struct {
	sinks []*Sink

	mu     sync.RWMutex
	closed bool
}

func NewHandler(sinks ...*Sink) *Handler {
	h := &Handler{
		sinks: sinks,
	}

	for _, sink := range h.sinks {
		sink.start()
	}

	return h
}

func (h *Handler) LogRecord(record types.Record, snapshotFields bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return
	}

	if snapshotFields {
		fields := make([]types.Field, len(record.Fields))
		copy(fields, record.Fields)

		record.Fields = fields
	}

	for _, sink := range h.sinks {
		if !sink.enabled(record.Level) {
			continue
		}

		sink.enqueue(record)
	}
}

func (h *Handler) Close() error {
	h.mu.Lock()

	if !h.closed {
		h.closed = true

		for _, sink := range h.sinks {
			sink.stop()
		}
	}

	h.mu.Unlock()

	var closeErr error

	for _, sink := range h.sinks {
		closeErr = errors.Join(
			closeErr,
			sink.wait(),
		)
	}

	return closeErr
}
