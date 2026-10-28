package types

import (
	"encoding/json"
	"strconv"

	keyslib "github.com/ygrebnov/keys"
)

type FieldKind int

const (
	FieldKindNull   FieldKind = 0
	FieldKindString FieldKind = 10
	FieldKindInt    FieldKind = 20
	FieldKindInt64  FieldKind = 24
	FieldKindBool   FieldKind = 60
	FieldKindError  FieldKind = 70
)

type Field interface {
	Kind() FieldKind
	Key() keyslib.Key
	String() string
	MarshalJSON() ([]byte, error)

	field()
}

func NewNullField(key keyslib.Key) Field {
	return nullField{key: key}
}

type nullField struct {
	key keyslib.Key
}

func (f nullField) Kind() FieldKind {
	return FieldKindNull
}

func (f nullField) Key() keyslib.Key {
	return f.key
}

func (f nullField) String() string {
	return "null"
}

func (f nullField) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

func (nullField) field() {}

func NewStringField(key keyslib.Key, value string) Field {
	return stringField{
		key:   key,
		value: value,
	}
}

type stringField struct {
	key   keyslib.Key
	value string
}

func (f stringField) Kind() FieldKind {
	return FieldKindString
}

func (f stringField) Key() keyslib.Key {
	return f.key
}

func (f stringField) String() string {
	return f.value
}

func (f stringField) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

func (f stringField) field() {}

func NewIntField(key keyslib.Key, value int) Field {
	return intField{
		key:   key,
		value: value,
	}
}

type intField struct {
	key   keyslib.Key
	value int
}

func (f intField) Kind() FieldKind {
	return FieldKindInt
}

func (f intField) Key() keyslib.Key {
	return f.key
}

func (f intField) String() string {
	return strconv.Itoa(f.value)
}

func (f intField) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

func (f intField) field() {}

func NewInt64Field(key keyslib.Key, value int64) Field {
	return int64Field{
		key:   key,
		value: value,
	}
}

type int64Field struct {
	key   keyslib.Key
	value int64
}

func (f int64Field) Kind() FieldKind {
	return FieldKindInt64
}

func (f int64Field) Key() keyslib.Key {
	return f.key
}

func (f int64Field) String() string {
	return strconv.FormatInt(f.value, 10)
}

func (f int64Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

func (f int64Field) field() {}

func NewBoolField(key keyslib.Key, value bool) Field {
	return boolField{
		key:   key,
		value: value,
	}
}

type boolField struct {
	key   keyslib.Key
	value bool
}

func (f boolField) Kind() FieldKind {
	return FieldKindBool
}

func (f boolField) Key() keyslib.Key {
	return f.key
}

func (f boolField) String() string {
	return strconv.FormatBool(f.value)
}

func (f boolField) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.value)
}

func (f boolField) field() {}

func NewErrField(key keyslib.Key, err error) Field {
	if err == nil {
		return NewNullField(key)
	}

	return NewStringField(key, err.Error())
}
