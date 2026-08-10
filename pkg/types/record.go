package types

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ygrebnov/errorc"
	keyslib "github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/errors"
	modellib "github.com/ygrebnov/model"
	modelliberrors "github.com/ygrebnov/model/pkg/errors"
	"github.com/ygrebnov/model/pkg/keys"
)

type Record struct {
	Time    time.Time `yaml:"time" validate:"nonZeroTime"`
	Level   Level     `yaml:"level" validate:"oneof(-8,-4,0,4,8,12)"`
	Message string    `yaml:"message" validate:"min(1)"`
	Fields  []Field   `yaml:"fields" validateElem:"dive,omitempty"`
}

type Field struct {
	Key   keyslib.Key `yaml:"key" validate:"min(1)"`
	Value string      `yaml:"value"`
}

var recordBinding *modellib.Binding[Record]
var validationRules []modellib.Rule

func init() {
	oneOfLevelRule, err := getOneOfUnderlyingIntRule[Level]()
	if err != nil {
		panic(err)
	}

	nonZeroTimeRule, err := getNonZeroTimeRule()
	if err != nil {
		panic(err)
	}

	minKeyRule, err := getMinKeyRule()
	if err != nil {
		panic(err)
	}

	validationRules = []modellib.Rule{
		oneOfLevelRule,
		nonZeroTimeRule,
		minKeyRule,
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

func getNonZeroTimeRule() (modellib.Rule, error) {
	return modellib.NewRule("nonZeroTime", func(v time.Time, _ ...string) error {
		if v.IsZero() {
			return errorc.With(
				modelliberrors.ErrRuleConstraintViolated,
				errorc.String(keys.RuleName, "nonZeroTime"),
			)
		}

		return nil
	})
}

func getMinKeyRule() (modellib.Rule, error) {
	return modellib.NewRule("min", func(k keyslib.Key, params ...string) error {
		if len(params) == 0 {
			return errorc.With(
				modelliberrors.ErrRuleMissingParameter,
				errorc.String(keys.RuleName, "min"),
			)
		}
		raw := strings.TrimSpace(params[0])
		v, err := strconv.ParseInt(raw, 10, 0)
		if err != nil {
			return errorc.With(
				modelliberrors.ErrRuleInvalidParameter,
				errorc.String(keys.RuleName, "min"),
				errorc.String(keys.RuleParamName, "length"),
				errorc.String(keys.RuleParamValue, raw),
				errorc.Error(keys.Cause, err),
			)
		}
		if len(k) < int(v) {
			return errorc.With(
				modelliberrors.ErrRuleConstraintViolated,
				errorc.String(keys.RuleName, "min"),
				errorc.String(keys.RuleParamName, "length"),
				errorc.String(keys.RuleParamValue, raw),
			)
		}

		return nil
	})
}

func (r Record) Validate(ctx context.Context) error {
	if err := recordBinding.Validate(ctx, &r); err != nil {
		return errorc.With(
			errors.ErrInvalidRecord,
			errorc.Error(keyslib.New("validation"), err),
		)
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
