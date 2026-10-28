package core

import (
	"bytes"
	"encoding/json"

	"github.com/ygrebnov/log/pkg/types"
)

type jsonFormatter struct {
	timestampFormat types.TimestampFormat
}

func (f jsonFormatter) Format(r types.Record) ([]byte, error) {
	var b bytes.Buffer

	b.WriteByte('{')

	if err := appendJSONField(&b, "_time", r.Time.Format(f.timestampFormat.Layout())); err != nil {
		return nil, err
	}

	b.WriteByte(',')

	if err := appendJSONField(&b, "_level", r.Level.String()); err != nil {
		return nil, err
	}

	b.WriteByte(',')

	if err := appendJSONField(&b, "_message", r.Message); err != nil {
		return nil, err
	}

	for _, field := range r.Fields {
		if field == nil {
			continue
		}

		key, err := json.Marshal(string(field.Key()))
		if err != nil {
			return nil, err
		}

		value, err := field.MarshalJSON()
		if err != nil {
			return nil, err
		}

		b.WriteByte(',')
		b.Write(key)
		b.WriteByte(':')
		b.Write(value)
	}

	b.WriteString("}\n")

	return b.Bytes(), nil
}

func appendJSONField(
	b *bytes.Buffer,
	key string,
	value string,
) error {
	encodedKey, err := json.Marshal(key)
	if err != nil {
		return err
	}

	encodedValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	b.Write(encodedKey)
	b.WriteByte(':')
	b.Write(encodedValue)

	return nil
}
