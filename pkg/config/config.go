package config

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"
	keyslib "github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/types"
	modellib "github.com/ygrebnov/model"
	modelliberrors "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

// Config is meant to be embedded into application config.
type Config struct {
	AppName string       `yaml:"app_name,omitempty"`
	Sinks   []SinkConfig `yaml:"sinks,omitempty" defaultElem:"dive" validateElem:"dive"`
}

type SinkConfig struct {
	Path          string        `yaml:"path,omitempty"`
	Kind          types.Kind    `yaml:"kind" default:"stderr" validate:"oneof(stdout,stderr,file)"` // +remote in v2
	Format        types.Format  `yaml:"format" default:"text" validate:"oneof(json,text)"`
	Level         types.Level   `yaml:"level" default:"0" validate:"oneof(-8,-4,0,4,8,12)"`
	QueueSize     int           `yaml:"queue_size" default:"1024" validate:"min(1)"`
	BufferSize    int           `yaml:"buffer_size" default:"65536" validate:"min(1)"`
	FlushInterval time.Duration `yaml:"flush_interval" default:"1s" validate:"nonZeroDuration"`
}

var binding *modellib.Binding[Config]
var validationRules []modellib.Rule

func init() {
	oneOfKindRule, err := getOneOfUnderlyingStringRule[types.Kind]()
	if err != nil {
		panic(err)
	}

	oneOfFormatRule, err := getOneOfUnderlyingStringRule[types.Format]()
	if err != nil {
		panic(err)
	}

	oneOfLevelRule, err := getOneOfUnderlyingIntRule[types.Level]()
	if err != nil {
		panic(err)
	}

	nonZeroDurationRule, err := getNonZeroDurationRule()
	if err != nil {
		panic(err)
	}

	validationRules = []modellib.Rule{
		oneOfKindRule,
		oneOfFormatRule,
		oneOfLevelRule,
		nonZeroDurationRule,
	}

	binding, err = modellib.NewBinding[Config](modellib.WithRules(validationRules...))
	if err != nil {
		panic(err)
	}
}

func GetValidationRules() []modellib.Rule {
	return validationRules
}

func getOneOfUnderlyingStringRule[T ~string]() (modellib.Rule, error) {
	return modellib.NewRule[T]("oneof", func(v T, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelliberrors.ErrRuleMissingParameter,
				errorc.String(keys.RuleName, "oneof"),
			)
		}
		for _, p := range params {
			if string(v) == p {
				return nil
			}
		}

		return errorc.With(
			modelliberrors.ErrRuleConstraintViolated,
			errorc.String(keys.RuleName, "oneof"),
			errorc.String(keys.RuleParamName, "allowed"),
			errorc.String(keys.RuleParamValue, strings.Join(params, ",")),
		)
	})
}

func getOneOfUnderlyingIntRule[T ~int]() (modellib.Rule, error) {
	return modellib.NewRule[T]("oneof", func(v T, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelliberrors.ErrRuleMissingParameter,
				errorc.String(keys.RuleName, "oneof"),
			)
		}
		for _, p := range params {
			paramValue, err := strconv.Atoi(p)
			if err != nil {
				return errorc.With(
					modelliberrors.ErrRuleInvalidParameter,
					errorc.String(keys.RuleName, "oneof"),
					errorc.String(keys.RuleParamName, "allowed"),
					errorc.String(keys.RuleParamValue, p),
				)
			}
			if int(v) == paramValue {
				return nil
			}
		}

		return errorc.With(
			modelliberrors.ErrRuleConstraintViolated,
			errorc.String(keys.RuleName, "oneof"),
			errorc.String(keys.RuleParamName, "allowed"),
			errorc.String(keys.RuleParamValue, strings.Join(params, ",")),
		)
	})
}

func getNonZeroDurationRule() (modellib.Rule, error) {
	return modellib.NewRule("nonZeroDuration", func(v time.Duration, _ ...string) error {
		if v < 1 {
			return errorc.With(
				modelliberrors.ErrRuleConstraintViolated,
				errorc.String(keys.RuleName, "nonZeroDuration"),
				errorc.String(keyslib.New("sink.flush.interval"), v.String()),
			)
		}

		return nil
	})
}

// ApplyDefaults sets defaults values for non-empty fields. Operation is idempotent, however may mutate the object.
func (c *Config) ApplyDefaults() error {
	return binding.ApplyDefaults(c)
}

func (c *Config) Validate(ctx context.Context) error {
	if err := binding.Validate(ctx, c); err != nil {
		return errorc.With(
			errors.ErrInvalidConfig,
			errorc.Error(keyslib.New("validation"), err),
		)
	}

	for _, sink := range c.Sinks {
		if c.AppName == "" && sink.Kind == types.KindFile && strings.TrimSpace(sink.Path) == "" {
			return errorc.With(
				errors.ErrInvalidConfig,
				errorc.String(keyslib.New("sink.kind"), string(sink.Kind)),
				errorc.String(keyslib.New("validation"), "path is required for file sink"),
			)
		}
	}
	return nil
}
