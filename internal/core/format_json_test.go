package core

import (
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/types"
)

func TestJSONFormatter_Format(t *testing.T) {
	at := time.Date(2026, 10, 28, 14, 0, 0, 123456789, time.UTC)

	const prefix = `{"_time":"2026-10-28T14:00:00.123456789Z","_level":`

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
			want:    prefix + `"info","_message":"operation completed"}` + "\n",
		},
		{
			name:    "empty fields",
			level:   types.LevelDebug,
			message: "debugging",
			fields:  []types.Field{},
			want:    prefix + `"debug","_message":"debugging"}` + "\n",
		},
		{
			name:    "string field",
			level:   types.LevelInfo,
			message: "user created",
			fields: []types.Field{
				types.NewStringField(keys.Key("user"), "alice"),
			},
			want: prefix + `"info","_message":"user created","user":"alice"}` + "\n",
		},
		{
			name:    "empty string",
			level:   types.LevelInfo,
			message: "empty value",
			fields: []types.Field{
				types.NewStringField(keys.Key("value"), ""),
			},
			want: prefix + `"info","_message":"empty value","value":""}` + "\n",
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
				`"info","_message":"escaped value","value":"hello \"world\"\\test\nnext"}` +
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
				`"info","_message":"numbers","count":42,"negative":-7,"large":9223372036854775807}` +
				"\n",
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
				`"info","_message":"flags","enabled":true,"disabled":false}` +
				"\n",
		},
		{
			name:    "null field",
			level:   types.LevelInfo,
			message: "null value",
			fields: []types.Field{
				types.NewNullField(keys.Key("value")),
			},
			want: prefix +
				`"info","_message":"null value","value":null}` +
				"\n",
		},
		{
			name:    "nil error",
			level:   types.LevelInfo,
			message: "no error",
			fields: []types.Field{
				types.NewErrField(keys.Key("error"), nil),
			},
			want: prefix +
				`"info","_message":"no error","error":null}` +
				"\n",
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
				`"error","_message":"operation failed","error":"connection refused"}` +
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
				`"info","_message":"payment completed","account":"ACC-123","amount":1500,"success":true,"error":null}` +
				"\n",
		},
		{
			name:    "duplicate keys preserved",
			level:   types.LevelInfo,
			message: "duplicates",
			fields: []types.Field{
				types.NewIntField(keys.Key("attempt"), 1),
				types.NewIntField(keys.Key("attempt"), 2),
			},
			want: prefix +
				`"info","_message":"duplicates","attempt":1,"attempt":2}` +
				"\n",
		},
		{
			name:    "metadata key collision preserved",
			level:   types.LevelInfo,
			message: "metadata collision",
			fields: []types.Field{
				types.NewStringField(keys.Key("_level"), "custom"),
				types.NewStringField(keys.Key("_message"), "custom message"),
				types.NewStringField(keys.Key("_time"), "custom time"),
			},
			want: prefix +
				`"info","_message":"metadata collision","_level":"custom","_message":"custom message","_time":"custom time"}` +
				"\n",
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
				`"info","_message":"nil field","first":"one","second":"two"}` +
				"\n",
		},
		{
			name:    "only nil fields",
			level:   types.LevelInfo,
			message: "nil fields",
			fields: []types.Field{
				nil,
				nil,
			},
			want: prefix + `"info","_message":"nil fields"}` + "\n",
		},
		{
			name:    "field key escaping",
			level:   types.LevelInfo,
			message: "keys",
			fields: []types.Field{
				types.NewStringField(keys.Key("user name"), "alice"),
				types.NewIntField(keys.Key("a=b"), 42),
				types.NewStringField(keys.Key("a\"b\\c"), "value"),
				types.NewIntField(keys.Key("line\nbreak"), 1),
			},
			want: prefix +
				`"info","_message":"keys","user name":"alice","a=b":42,"a\"b\\c":"value","line\nbreak":1}` +
				"\n",
		},
		{
			name:    "message escaping",
			level:   types.LevelWarn,
			message: "first\nsecond\rthird\t\"quoted\"\\path",
			want: prefix +
				`"warn","_message":"first\nsecond\rthird\t\"quoted\"\\path"}` +
				"\n",
		},
		{
			name:    "control character escaping",
			level:   types.LevelInfo,
			message: "before\x01after",
			fields: []types.Field{
				types.NewStringField(keys.Key("value"), "a\x00b"),
			},
			want: prefix +
				`"info","_message":"before\u0001after","value":"a\u0000b"}` +
				"\n",
		},
		{
			name:    "html escaping",
			level:   types.LevelInfo,
			message: "<script>&</script>",
			fields: []types.Field{
				types.NewStringField(keys.Key("value"), "<>&"),
			},
			want: prefix +
				`"info","_message":"\u003cscript\u003e\u0026\u003c/script\u003e","value":"\u003c\u003e\u0026"}` +
				"\n",
		},
		{
			name:    "trace level",
			level:   types.LevelTrace,
			message: "trace",
			want:    prefix + `"trace","_message":"trace"}` + "\n",
		},
		{
			name:    "fatal level",
			level:   types.LevelFatal,
			message: "fatal",
			want:    prefix + `"fatal","_message":"fatal"}` + "\n",
		},
	}

	formatter := jsonFormatter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := types.NewRecord(
				at,
				tt.level,
				tt.message,
				tt.fields...,
			)

			got, err := formatter.Format(record)
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
