package core

import (
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/types"
)

func TestTextFormatter_Format(t *testing.T) {
	at := time.Date(2026, 10, 28, 14, 0, 0, 123456789, time.UTC)
	const prefix = "2026-10-28T14:00:00.123456789Z "

	tests := []struct {
		name    string
		level   types.Level
		message string
		fields  []types.Field
		want    string
	}{
		{
			name:    "no fields",
			level:   types.LevelInfo,
			message: "operation completed",
			want:    prefix + "INFO operation completed\n",
		},
		{
			name:    "empty fields",
			level:   types.LevelDebug,
			message: "debugging",
			fields:  []types.Field{},
			want:    prefix + "DEBUG debugging\n",
		},
		{
			name:    "string field",
			level:   types.LevelInfo,
			message: "user created",
			fields: []types.Field{
				types.NewStringField(keys.Key("user"), "alice"),
			},
			want: prefix + `INFO user created, user="alice"` + "\n",
		},
		{
			name:    "empty string",
			level:   types.LevelInfo,
			message: "empty value",
			fields: []types.Field{
				types.NewStringField(keys.Key("value"), ""),
			},
			want: prefix + `INFO empty value, value=""` + "\n",
		},
		{
			name:    "string escaping",
			level:   types.LevelInfo,
			message: "escaped value",
			fields: []types.Field{
				types.NewStringField(
					keys.Key("value"),
					"hello \"world\"\\test\nnext",
				),
			},
			want: prefix +
				`INFO escaped value, value="hello \"world\"\\test\nnext"` +
				"\n",
		},
		{
			name:    "integer fields",
			level:   types.LevelInfo,
			message: "numbers",
			fields: []types.Field{
				types.NewIntField(keys.Key("count"), 42),
				types.NewIntField(keys.Key("negative"), -7),
				types.NewInt64Field(keys.Key("large"), 9223372036854775807),
			},
			want: prefix +
				"INFO numbers, count=42 negative=-7 large=9223372036854775807\n",
		},
		{
			name:    "boolean fields",
			level:   types.LevelInfo,
			message: "flags",
			fields: []types.Field{
				types.NewBoolField(keys.Key("enabled"), true),
				types.NewBoolField(keys.Key("disabled"), false),
			},
			want: prefix +
				"INFO flags, enabled=true disabled=false\n",
		},
		{
			name:    "null field",
			level:   types.LevelInfo,
			message: "null value",
			fields: []types.Field{
				types.NewNullField(keys.Key("value")),
			},
			want: prefix + "INFO null value, value=null\n",
		},
		{
			name:    "nil error",
			level:   types.LevelInfo,
			message: "no error",
			fields: []types.Field{
				types.NewErrField(keys.Key("error"), nil),
			},
			want: prefix + "INFO no error, error=null\n",
		},
		{
			name:    "non-nil error",
			level:   types.LevelError,
			message: "operation failed",
			fields: []types.Field{
				types.NewErrField(
					keys.Key("error"),
					errors.New("connection refused"),
				),
			},
			want: prefix +
				`ERROR operation failed, error="connection refused"` +
				"\n",
		},
		{
			name:    "mixed fields",
			level:   types.LevelInfo,
			message: "payment completed",
			fields: []types.Field{
				types.NewStringField(keys.Key("account"), "ACC-123"),
				types.NewInt64Field(keys.Key("amount"), 1500),
				types.NewBoolField(keys.Key("success"), true),
				types.NewNullField(keys.Key("error")),
			},
			want: prefix +
				`INFO payment completed, account="ACC-123" amount=1500 success=true error=null` +
				"\n",
		},
		{
			name:    "duplicate keys",
			level:   types.LevelInfo,
			message: "duplicates",
			fields: []types.Field{
				types.NewIntField(keys.Key("attempt"), 1),
				types.NewIntField(keys.Key("attempt"), 2),
			},
			want: prefix +
				"INFO duplicates, attempt=1 attempt=2\n",
		},
		{
			name:    "nil field ignored",
			level:   types.LevelInfo,
			message: "nil field",
			fields: []types.Field{
				types.NewStringField(keys.Key("first"), "one"),
				nil,
				types.NewStringField(keys.Key("second"), "two"),
			},
			want: prefix +
				`INFO nil field, first="one" second="two"` +
				"\n",
		},
		{
			name:    "field key with spaces",
			level:   types.LevelInfo,
			message: "keys",
			fields: []types.Field{
				types.NewStringField(keys.Key("user name"), "alice"),
			},
			want: prefix +
				`INFO keys, "user name"="alice"` +
				"\n",
		},
		{
			name:    "field key with equals sign",
			level:   types.LevelInfo,
			message: "keys",
			fields: []types.Field{
				types.NewIntField(keys.Key("a=b"), 42),
			},
			want: prefix +
				`INFO keys, "a=b"=42` +
				"\n",
		},
		{
			name:    "field key with quotes and backslash",
			level:   types.LevelInfo,
			message: "keys",
			fields: []types.Field{
				types.NewStringField(keys.Key("a\"b\\c"), "value"),
			},
			want: prefix +
				`INFO keys, "a\"b\\c"="value"` +
				"\n",
		},
		{
			name:    "field key with newline",
			level:   types.LevelInfo,
			message: "keys",
			fields: []types.Field{
				types.NewIntField(keys.Key("line\nbreak"), 1),
			},
			want: prefix +
				`INFO keys, "line\nbreak"=1` +
				"\n",
		},
		{
			name:    "message with newline",
			level:   types.LevelInfo,
			message: "first\nsecond",
			want:    prefix + `INFO first\nsecond` + "\n",
		},
		{
			name:    "message with carriage return and tab",
			level:   types.LevelWarn,
			message: "first\rsecond\tthird",
			want:    prefix + `WARN first\rsecond\tthird` + "\n",
		},
		{
			name:    "message with control character",
			level:   types.LevelInfo,
			message: "before\x01after",
			want:    prefix + `INFO before\x01after` + "\n",
		},
		{
			name:    "ordinary message preserves quotes",
			level:   types.LevelInfo,
			message: `operation "completed"`,
			want:    prefix + `INFO operation "completed"` + "\n",
		},
		{
			name:    "trace level",
			level:   types.LevelTrace,
			message: "trace",
			want:    prefix + "TRACE trace\n",
		},
		{
			name:    "fatal level",
			level:   types.LevelFatal,
			message: "fatal",
			want:    prefix + "FATAL fatal\n",
		},
		{
			name:    "only nil fields",
			level:   types.LevelInfo,
			message: "nil fields",
			fields: []types.Field{
				nil,
				nil,
			},
			want: prefix + "INFO nil fields\n",
		},
	}

	f := textFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := types.NewRecord(
				at,
				tt.level,
				tt.message,
				tt.fields...,
			)

			got, err := f.Format(record)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			if string(got) != tt.want {
				t.Errorf(
					"Format() mismatch:\n got: %q\nwant: %q",
					string(got),
					tt.want,
				)
			}
		})
	}
}
