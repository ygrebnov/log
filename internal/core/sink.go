package core

import (
	"bufio"
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/log/internal/fs"
	"github.com/ygrebnov/log/pkg/config"
	errorspkg "github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/keys"
	"github.com/ygrebnov/log/pkg/types"
)

type Sink struct {
	records chan types.Record
	done    chan struct{}

	level     types.Level
	formatter formatter

	writer *bufio.Writer
	closer closer

	flushInterval time.Duration

	startOnce sync.Once
	stopOnce  sync.Once

	queueMu sync.RWMutex
	stopped bool

	errMu sync.Mutex
	err   error
}

type closer interface {
	Close() error
}

func NewSink(
	ctx context.Context,
	cfg *config.SinkConfig,
	appName string,
) (*Sink, error) {
	var (
		out io.Writer
		c   closer
	)

	var f formatter

	switch cfg.Format {
	case types.FormatText:
		f = textFormatter{
			timestampFormat: cfg.TimestampFormat,
		}

	case types.FormatJSON:
		f = jsonFormatter{
			timestampFormat: cfg.TimestampFormat,
		}

	default:
		return nil, errorc.With(
			errorspkg.ErrInvalidSinkFormat,
			errorc.String(keys.SinkFormat, string(cfg.Format)),
		)
	}

	switch cfg.Kind {
	case types.KindStdOut:
		out = os.Stdout

	case types.KindStdErr:
		out = os.Stderr

	case types.KindFile:
		w, fileCloser, err := fs.OpenWriter(cfg, appName)
		if err != nil {
			return nil, errorc.With(
				errorspkg.ErrCannotOpenFileWriter,
				errorc.Error(keys.Cause, err),
			)
		}

		out = w
		c = fileCloser

	default:
		return nil, errorc.With(
			errorspkg.ErrInvalidSinkKind,
			errorc.String(keys.SinkKind, string(cfg.Kind)),
		)
	}

	// ctx is intentionally a constructor-scoped context.
	// It will be used by the remote sink during initialization.
	_ = ctx

	return &Sink{
		records:       make(chan types.Record, cfg.QueueSize),
		done:          make(chan struct{}),
		level:         cfg.Level,
		formatter:     f,
		writer:        bufio.NewWriterSize(out, cfg.BufferSize),
		closer:        c,
		flushInterval: cfg.FlushInterval,
	}, nil
}

func (s *Sink) enabled(level types.Level) bool {
	return level >= s.level
}

func (s *Sink) enqueue(r types.Record) {
	s.queueMu.RLock()
	defer s.queueMu.RUnlock()

	if s.stopped {
		return
	}

	select {
	case s.records <- r:
	default:
		// Queue full: drop the record.
	}
}

func (s *Sink) start() {
	s.startOnce.Do(func() {
		go s.run()
	})
}

func (s *Sink) stop() {
	s.stopOnce.Do(func() {
		s.queueMu.Lock()
		defer s.queueMu.Unlock()

		s.stopped = true
		close(s.records)
	})
}

func (s *Sink) wait() error {
	<-s.done

	return s.getError()
}

func (s *Sink) Close() error {
	s.start()
	s.stop()

	return s.wait()
}

func (s *Sink) run() {
	defer close(s.done)

	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case r, ok := <-s.records:
			if !ok {
				if s.getError() == nil {
					s.flush()
				}

				s.closeOutput()
				return
			}

			s.write(r)

		case <-ticker.C:
			if s.getError() == nil {
				s.flush()
			}
		}
	}
}

func (s *Sink) write(r types.Record) {
	if s.getError() != nil {
		return
	}

	data, err := s.formatter.Format(r)
	if err != nil {
		// Skip this record, but keep the sink operational.
		return
	}

	if _, errWrite := s.writer.Write(data); errWrite != nil {
		s.setError(
			errorc.With(
				errorspkg.ErrCannotWriteLogRecord,
				errorc.Error(keys.Cause, errWrite),
			),
		)
	}
}

func (s *Sink) flush() {
	if err := s.writer.Flush(); err != nil {
		s.setError(
			errorc.With(
				errorspkg.ErrCannotFlushWriter,
				errorc.Error(keys.Cause, err),
			),
		)
	}
}

func (s *Sink) closeOutput() {
	if s.closer == nil {
		return
	}

	if err := s.closer.Close(); err != nil {
		s.setError(err)
	}
}

func (s *Sink) setError(err error) {
	if err == nil {
		return
	}

	s.errMu.Lock()
	defer s.errMu.Unlock()

	if s.err == nil {
		s.err = err
	}
}

func (s *Sink) getError() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	return s.err
}
