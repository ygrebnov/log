package types

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/log/pkg/errors"
	modellib "github.com/ygrebnov/model"
	modelliberrors "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

type Record struct {
	Time    time.Time `yaml:"time"`
	Level   Level     `yaml:"level" validate:"oneof(-8,-4,0,4,8,12)"`
	Message string    `yaml:"message" validate:"min(1)"`
	Fields  []Field   `yaml:"fields"`
}

var recordBinding *modellib.Binding[Record]
var validationRules []modellib.Rule

func init() {
	oneOfLevelRule, err := getOneOfUnderlyingIntRule[Level]()
	if err != nil {
		panic(err)
	}

	validationRules = []modellib.Rule{
		oneOfLevelRule,
	}

	recordBinding, err = modellib.NewBinding[Record](modellib.WithRules(validationRules...))
	if err != nil {
		panic(err)
	}
}

func GetValidationRules() []modellib.Rule {
	rules := make([]modellib.Rule, len(validationRules))
	copy(rules, validationRules)

	return rules
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

func (r Record) Validate(ctx context.Context) error {
	if err := recordBinding.Validate(ctx, &r); err != nil {
		return errorc.With(
			errors.ErrInvalidRecord,
			errorc.Error("validation", err),
		)
	}

	if r.Time.IsZero() {
		return errorc.With(
			errors.ErrInvalidRecord,
			errorc.String("validation", "time must not be zero"),
		)
	}

	for i, field := range r.Fields {
		if field == nil || len(field.Key()) == 0 {
			return errorc.With(
				errors.ErrInvalidRecord,
				errorc.Int("field_index", i),
				errorc.String(
					"validation",
					"field must not be nil and must have a non-empty key",
				),
			)
		}
	}

	return nil
}

func NewRecord(
	at time.Time,
	level Level,
	message string,
	fields ...Field,
) Record {
	fieldsCopy := make([]Field, len(fields))
	copy(fieldsCopy, fields)

	return Record{
		Time:    at,
		Level:   level,
		Message: message,
		Fields:  fieldsCopy,
	}
}
