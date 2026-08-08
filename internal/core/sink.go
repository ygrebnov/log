package core

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
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
	records chan record
	done    chan struct{}

	level types.Level

	writer *bufio.Writer
	closer closer

	flushInterval time.Duration

	startOnce sync.Once
	stopOnce  sync.Once

	errMu sync.Mutex
	err   error
}

type closer interface {
	Close() error
}

type record struct {
	time    time.Time
	level   types.Level
	message string
	fields  []types.Field
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
		records:       make(chan record, cfg.QueueSize),
		done:          make(chan struct{}),
		level:         cfg.Level,
		writer:        bufio.NewWriterSize(out, cfg.BufferSize),
		closer:        c,
		flushInterval: cfg.FlushInterval,
	}, nil
}

func (s *Sink) enabled(level types.Level) bool {
	return level >= s.level
}

func (s *Sink) enqueue(r record) {
	s.records <- r
}

func (s *Sink) start() {
	s.startOnce.Do(func() {
		go s.run()
	})
}

func (s *Sink) stop() {
	s.stopOnce.Do(func() {
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
				s.flush()
				s.closeOutput()
				return
			}

			s.write(r)

		case <-ticker.C:
			s.flush()
		}
	}
}

func (s *Sink) write(r record) {
	if s.getError() != nil {
		return
	}

	var b strings.Builder

	b.WriteString(r.time.Format(time.RFC3339Nano))
	b.WriteByte(' ')
	b.WriteString(r.level.String())
	b.WriteByte(' ')
	b.WriteString(r.message)

	for _, field := range r.fields {
		if field.Key == "" && field.Value == "" {
			continue
		}

		b.WriteByte(' ')

		if field.Key != "" {
			b.WriteString(string(field.Key))
			b.WriteByte('=')
		}

		b.WriteString(strconv.Quote(field.Value))
	}

	b.WriteByte('\n')

	if _, err := s.writer.WriteString(b.String()); err != nil {
		s.setError(
			errorc.With(
				errorspkg.ErrCannotWriteLogRecord,
				errorc.Error(keys.Cause, err),
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

	s.err = errors.Join(s.err, err)
}

func (s *Sink) getError() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	return s.err
}
