package types

import (
	"context"
	"sync"
	"time"

	"github.com/ygrebnov/errorc"
	"github.com/ygrebnov/log/pkg/errors"
	"github.com/ygrebnov/log/pkg/keys"
	modellib "github.com/ygrebnov/model"
)

type Record struct {
	Time    time.Time `yaml:"time"`
	Level   Level     `yaml:"level" validate:"oneof(-8,-4,0,4,8,12)"`
	Message string    `yaml:"message" validate:"min(1)"`
	Fields  []Field   `yaml:"fields"`
}

var (
	recordBinding    *modellib.Binding[Record]
	recordBindingErr error
	newRecordBinding sync.Once
)

func getRecordBinding() (*modellib.Binding[Record], error) {
	newRecordBinding.Do(func() {
		recordBinding, recordBindingErr = modellib.NewBinding[Record]()
	})

	return recordBinding, recordBindingErr
}

func (r Record) Validate(ctx context.Context) error {
	b, err := getRecordBinding()
	if err != nil {
		return err
	}

	if errValidate := b.Validate(ctx, &r); errValidate != nil {
		return errorc.WithCause(errors.ErrInvalidRecord, errValidate)
	}

	if r.Time.IsZero() {
		return errorc.With(
			errors.ErrInvalidRecord,
			errorc.String(keys.Validation, "time must not be zero"),
		)
	}

	for i, field := range r.Fields {
		if field == nil || len(field.Key()) == 0 {
			return errorc.With(
				errors.ErrInvalidRecord,
				errorc.Int("field_index", i),
				errorc.String(
					keys.Validation,
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
