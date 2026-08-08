package core

import (
	"errors"
	"sync"
	"time"

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

func (h *Handler) Log(level types.Level, message string, fields ...types.Field) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return
	}

	fieldsCopy := make([]types.Field, len(fields))
	copy(fieldsCopy, fields)

	r := record{
		time:    time.Now(),
		level:   level,
		message: message,
		fields:  fieldsCopy,
	}

	for _, sink := range h.sinks {
		if !sink.enabled(level) {
			continue
		}

		sink.enqueue(r)
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
