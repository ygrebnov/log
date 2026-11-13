package config

import (
	"context"
	nativeerrors "errors"
	"strings"
	"sync"
	"time"

	"github.com/ygrebnov/errorc"
	keyslib "github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/types"
	modellib "github.com/ygrebnov/model"
)

// Config is meant to be embedded into application config.
type Config struct {
	AppName string       `yaml:"app_name,omitempty" env:"APP_NAME"`
	Sinks   []SinkConfig `yaml:"sinks,omitempty" defaultElem:"dive" validateElem:"dive"`
}

type SinkConfig struct {
	Path            string                `yaml:"path,omitempty"`
	Kind            types.Kind            `yaml:"kind" default:"stderr" validate:"oneof(stdout,stderr,file)"` // +remote in v2
	Format          types.Format          `yaml:"format" default:"text" validate:"oneof(json,text)"`
	Level           types.Level           `yaml:"level" default:"0" validate:"oneof(-8,-4,0,4,8,12)"`
	TimestampFormat types.TimestampFormat `yaml:"timestamp_format" env:"TIMESTAMP_FORMAT" default:"rfc3339nano" validate:"oneof(rfc3339nano,rfc3339)"`
	QueueSize       int                   `yaml:"queue_size" env:"QUEUE_SIZE" default:"1024" validate:"min(1)"`
	BufferSize      int                   `yaml:"buffer_size" env:"BUFFER_SIZE" default:"65536" validate:"min(1)"`
	FlushInterval   time.Duration         `yaml:"flush_interval" env:"FLUSH_INTERVAL" default:"1s" validate:"min(1ns)"`
}

var (
	binding    *modellib.Binding[Config]
	bindingErr error
	newBinding sync.Once
)

func getBinding() (*modellib.Binding[Config], error) {
	newBinding.Do(func() {
		binding, bindingErr = modellib.NewBinding[Config]()
	})
	return binding, bindingErr
}

// ApplyDefaults sets defaults values for non-empty fields.
// Operation is idempotent, however may mutate the object.
func (c *Config) ApplyDefaults() error {
	b, err := getBinding()
	if err != nil {
		return err
	}

	return b.ApplyDefaults(c)
}

func (c *Config) Validate(ctx context.Context) error {
	b, err := getBinding()
	if err != nil {
		return err
	}

	if errValidate := b.Validate(ctx, c); errValidate != nil {
		return nativeerrors.Join(
			errors.ErrInvalidConfig,
			errValidate,
		)
	}

	for _, sink := range c.Sinks {
		if c.AppName == "" && sink.Kind == types.KindFile && strings.TrimSpace(sink.Path) == "" {
			return errorc.With(
				errors.ErrInvalidConfig,
				errorc.String(keyslib.New("sink.kind"), string(sink.Kind)),
				errorc.String(
					keyslib.New("validation"),
					"app_name or path is required for file sink",
				),
			)
		}
	}

	return nil
}
