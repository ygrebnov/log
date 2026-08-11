package core

import (
	"errors"
	"sync"

	"github.com/ygrebnov/log/pkg/types"
)

type handlerState struct {
	sinks []*Sink

	mu     sync.RWMutex
	closed bool
}

type Handler struct {
	state  *handlerState
	fields []types.Field // optional fields to be added to every log record
}

func NewHandler(sinks ...*Sink) *Handler {
	state := &handlerState{
		sinks: sinks,
	}

	for _, sink := range sinks {
		sink.start()
	}

	return &Handler{
		state: state,
	}
}

func (h *Handler) LogRecord(record types.Record, copyFields bool) {
	h.state.mu.RLock()
	defer h.state.mu.RUnlock()

	if h.state.closed {
		return
	}

	if copyFields || len(h.fields) > 0 {
		fields := make([]types.Field, 0, len(h.fields)+len(record.Fields))
		fields = append(fields, h.fields...)
		fields = append(fields, record.Fields...)
		record.Fields = fields
	}

	for _, sink := range h.state.sinks {
		if sink.enabled(record.Level) {
			sink.enqueue(record)
		}
	}
}

func (h *Handler) Close() error {
	h.state.mu.Lock()

	if !h.state.closed {
		h.state.closed = true

		for _, sink := range h.state.sinks {
			sink.stop()
		}
	}

	h.state.mu.Unlock()

	var closeErr error

	for _, sink := range h.state.sinks {
		closeErr = errors.Join(
			closeErr,
			sink.wait(),
		)
	}

	return closeErr
}

func (h *Handler) With(fields ...types.Field) *Handler {
	h.state.mu.RLock()
	defer h.state.mu.RUnlock()

	if h.state.closed {
		return h
	}

	newFields := make([]types.Field, len(h.fields)+len(fields))
	copy(newFields, h.fields)
	copy(newFields[len(h.fields):], fields)

	return &Handler{
		state:  h.state,
		fields: newFields,
	}
}
