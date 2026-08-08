package config_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ygrebnov/log/pkg/config"
	logerrors "github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/types"
)

func TestConfig_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want config.Config
	}{
		{
			name: "empty config",
			cfg:  config.Config{},
			want: config.Config{},
		},
		{
			name: "empty sink",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					{},
				},
			},
			want: config.Config{
				Sinks: []config.SinkConfig{
					{
						Kind:          types.KindStdErr,
						Format:        types.FormatJSON,
						Level:         types.LevelInfo,
						QueueSize:     1024,
						BufferSize:    65536,
						FlushInterval: time.Second,
					},
				},
			},
		},
		{
			name: "partial sink",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{
						Kind:  types.KindStdOut,
						Level: types.LevelDebug,
					},
				},
			},
			want: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{
						Kind:          types.KindStdOut,
						Format:        types.FormatJSON,
						Level:         types.LevelDebug,
						QueueSize:     1024,
						BufferSize:    65536,
						FlushInterval: time.Second,
					},
				},
			},
		},
		{
			name: "explicit values preserved",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{
						Path:          "/tmp/application.log",
						Kind:          types.KindFile,
						Format:        types.FormatText,
						Level:         types.LevelTrace,
						QueueSize:     64,
						BufferSize:    4096,
						FlushInterval: 250 * time.Millisecond,
					},
				},
			},
			want: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{
						Path:          "/tmp/application.log",
						Kind:          types.KindFile,
						Format:        types.FormatText,
						Level:         types.LevelTrace,
						QueueSize:     64,
						BufferSize:    4096,
						FlushInterval: 250 * time.Millisecond,
					},
				},
			},
		},
		{
			name: "multiple sinks defaulted independently",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{},
					{
						Kind:   types.KindStdOut,
						Format: types.FormatText,
						Level:  types.LevelWarn,
					},
					{
						Path: "/tmp/application.log",
						Kind: types.KindFile,
					},
				},
			},
			want: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{
						Kind:          types.KindStdErr,
						Format:        types.FormatJSON,
						Level:         types.LevelInfo,
						QueueSize:     1024,
						BufferSize:    65536,
						FlushInterval: time.Second,
					},
					{
						Kind:          types.KindStdOut,
						Format:        types.FormatText,
						Level:         types.LevelWarn,
						QueueSize:     1024,
						BufferSize:    65536,
						FlushInterval: time.Second,
					},
					{
						Path:          "/tmp/application.log",
						Kind:          types.KindFile,
						Format:        types.FormatJSON,
						Level:         types.LevelInfo,
						QueueSize:     1024,
						BufferSize:    65536,
						FlushInterval: time.Second,
					},
				},
			},
		},
		{
			name: "invalid explicit values are not replaced",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					{
						Kind:          types.Kind("invalid"),
						Format:        types.Format("invalid"),
						Level:         types.Level(100),
						QueueSize:     -1,
						BufferSize:    -1,
						FlushInterval: -time.Second,
					},
				},
			},
			want: config.Config{
				Sinks: []config.SinkConfig{
					{
						Kind:          types.Kind("invalid"),
						Format:        types.Format("invalid"),
						Level:         types.Level(100),
						QueueSize:     -1,
						BufferSize:    -1,
						FlushInterval: -time.Second,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg

			if err := cfg.ApplyDefaults(); err != nil {
				t.Fatalf("ApplyDefaults() error: %v", err)
			}

			if !reflect.DeepEqual(cfg, tt.want) {
				t.Fatalf(
					"ApplyDefaults() result mismatch\n got: %#v\nwant: %#v",
					cfg,
					tt.want,
				)
			}

			// Applying defaults must be idempotent.
			first := cfg

			if err := cfg.ApplyDefaults(); err != nil {
				t.Fatalf("second ApplyDefaults() error: %v", err)
			}

			if !reflect.DeepEqual(cfg, first) {
				t.Fatalf(
					"ApplyDefaults() is not idempotent\nfirst: %#v\nsecond: %#v",
					first,
					cfg,
				)
			}
		})
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "empty config",
			cfg:  config.Config{},
		},
		{
			name: "stdout",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					validSink(types.KindStdOut),
				},
			},
		},
		{
			name: "stderr",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					validSink(types.KindStdErr),
				},
			},
		},
		{
			name: "file with explicit path and no app name",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindFile)
						sink.Path = "/tmp/application.log"
						return sink
					}(),
				},
			},
		},
		{
			name: "file without path with app name",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					validSink(types.KindFile),
				},
			},
		},
		{
			name: "file with whitespace path with app name",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindFile)
						sink.Path = "   "
						return sink
					}(),
				},
			},
		},
		{
			name: "file path with surrounding whitespace",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindFile)
						sink.Path = "  /tmp/application.log  "
						return sink
					}(),
				},
			},
		},
		{
			name: "text format",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Format = types.FormatText
						return sink
					}(),
				},
			},
		},
		{
			name: "minimum queue and buffer sizes",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.QueueSize = 1
						sink.BufferSize = 1
						return sink
					}(),
				},
			},
		},
		{
			name: "minimum positive flush interval",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.FlushInterval = time.Nanosecond
						return sink
					}(),
				},
			},
		},
		{
			name: "all sink kinds",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					validSink(types.KindStdOut),
					validSink(types.KindStdErr),
					validSink(types.KindFile),
				},
			},
		},
	}

	for _, level := range []types.Level{
		types.LevelTrace,
		types.LevelDebug,
		types.LevelInfo,
		types.LevelWarn,
		types.LevelError,
		types.LevelFatal,
	} {
		tests = append(tests, struct {
			name string
			cfg  config.Config
		}{
			name: "level_" + level.String(),
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Level = level
						return sink
					}(),
				},
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(context.Background()); err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "zero valued sink without defaults",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					{},
				},
			},
		},
		{
			name: "invalid kind",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Kind = types.Kind("unknown")
						return sink
					}(),
				},
			},
		},
		{
			name: "remote kind not supported in v1",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					validSink(types.KindRemote),
				},
			},
		},
		{
			name: "invalid format",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Format = types.Format("xml")
						return sink
					}(),
				},
			},
		},
		{
			name: "invalid level below trace",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Level = types.Level(-9)
						return sink
					}(),
				},
			},
		},
		{
			name: "invalid level between debug and info",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Level = types.Level(-1)
						return sink
					}(),
				},
			},
		},
		{
			name: "invalid level above fatal",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.Level = types.Level(13)
						return sink
					}(),
				},
			},
		},
		{
			name: "zero queue size",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.QueueSize = 0
						return sink
					}(),
				},
			},
		},
		{
			name: "negative queue size",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.QueueSize = -1
						return sink
					}(),
				},
			},
		},
		{
			name: "zero buffer size",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.BufferSize = 0
						return sink
					}(),
				},
			},
		},
		{
			name: "negative buffer size",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.BufferSize = -1
						return sink
					}(),
				},
			},
		},
		{
			name: "zero flush interval",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.FlushInterval = 0
						return sink
					}(),
				},
			},
		},
		{
			name: "negative flush interval",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindStdOut)
						sink.FlushInterval = -time.Nanosecond
						return sink
					}(),
				},
			},
		},
		{
			name: "file without path or app name",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					validSink(types.KindFile),
				},
			},
		},
		{
			name: "file with whitespace path and no app name",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					func() config.SinkConfig {
						sink := validSink(types.KindFile)
						sink.Path = "   "
						return sink
					}(),
				},
			},
		},
		{
			name: "one invalid sink among valid sinks",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					validSink(types.KindStdOut),
					func() config.SinkConfig {
						sink := validSink(types.KindStdErr)
						sink.QueueSize = 0
						return sink
					}(),
					validSink(types.KindFile),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(context.Background())
			if err == nil {
				t.Fatal("Validate() expected error, got nil")
			}

			if !errors.Is(err, logerrors.ErrInvalidConfig) {
				t.Fatalf(
					"Validate() error = %v, want errors.Is(err, ErrInvalidConfig)",
					err,
				)
			}
		})
	}
}

func TestConfig_ApplyDefaultsThenValidate(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "empty config",
			cfg:  config.Config{},
		},
		{
			name: "default stdout-compatible sink becomes valid stderr sink",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					{},
				},
			},
		},
		{
			name: "partial stdout sink",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					{
						Kind: types.KindStdOut,
					},
				},
			},
		},
		{
			name: "file sink with app name",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{
						Kind: types.KindFile,
					},
				},
			},
		},
		{
			name: "file sink with explicit path",
			cfg: config.Config{
				Sinks: []config.SinkConfig{
					{
						Kind: types.KindFile,
						Path: "/tmp/application.log",
					},
				},
			},
		},
		{
			name: "multiple partially configured sinks",
			cfg: config.Config{
				AppName: "application",
				Sinks: []config.SinkConfig{
					{},
					{
						Kind:  types.KindStdOut,
						Level: types.LevelDebug,
					},
					{
						Kind:   types.KindFile,
						Format: types.FormatText,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg

			if err := cfg.ApplyDefaults(); err != nil {
				t.Fatalf("ApplyDefaults() error: %v", err)
			}

			if err := cfg.Validate(context.Background()); err != nil {
				t.Fatalf("Validate() after ApplyDefaults() error: %v", err)
			}
		})
	}
}

func validSink(kind types.Kind) config.SinkConfig {
	return config.SinkConfig{
		Kind:          kind,
		Format:        types.FormatJSON,
		Level:         types.LevelInfo,
		QueueSize:     1024,
		BufferSize:    65536,
		FlushInterval: time.Second,
	}
}
