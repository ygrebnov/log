package types

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ygrebnov/keys"
	logerrors "github.com/ygrebnov/log/pkg/errors"
)

func TestRecord(t *testing.T) {
	now := time.Date(2026, time.August, 10, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		at      time.Time
		level   Level
		message string
		fields  []Field
		wantErr bool
	}{
		{
			name:    "valid without fields",
			at:      now,
			level:   LevelInfo,
			message: "message",
		},
		{
			name:    "valid with field",
			at:      now,
			level:   LevelInfo,
			message: "message",
			fields: []Field{
				stringField{
					key:   keys.New("key"),
					value: "value",
				},
			},
		},
		{
			name:    "valid with multiple fields",
			at:      now,
			level:   LevelWarn,
			message: "message",
			fields: []Field{
				stringField{
					key:   keys.New("first"),
					value: "one",
				},
				stringField{
					key:   keys.New("second"),
					value: "two",
				},
			},
		},
		{
			name:    "valid empty field value",
			at:      now,
			level:   LevelInfo,
			message: "message",
			fields: []Field{
				stringField{
					key:   keys.New("key"),
					value: "",
				},
			},
		},
		{
			name:    "valid one character message",
			at:      now,
			level:   LevelInfo,
			message: "x",
		},
		{
			name:    "valid trace level",
			at:      now,
			level:   LevelTrace,
			message: "message",
		},
		{
			name:    "valid debug level",
			at:      now,
			level:   LevelDebug,
			message: "message",
		},
		{
			name:    "valid info level",
			at:      now,
			level:   LevelInfo,
			message: "message",
		},
		{
			name:    "valid warn level",
			at:      now,
			level:   LevelWarn,
			message: "message",
		},
		{
			name:    "valid error level",
			at:      now,
			level:   LevelError,
			message: "message",
		},
		{
			name:    "valid fatal level",
			at:      now,
			level:   LevelFatal,
			message: "message",
		},
		{
			name:    "zero time",
			at:      time.Time{},
			level:   LevelInfo,
			message: "message",
			wantErr: true,
		},
		{
			name:    "empty message",
			at:      now,
			level:   LevelInfo,
			message: "",
			wantErr: true,
		},
		{
			name:    "invalid level below trace",
			at:      now,
			level:   Level(-9),
			message: "message",
			wantErr: true,
		},
		{
			name:    "invalid level between trace and debug",
			at:      now,
			level:   Level(-7),
			message: "message",
			wantErr: true,
		},
		{
			name:    "invalid level between debug and info",
			at:      now,
			level:   Level(-1),
			message: "message",
			wantErr: true,
		},
		{
			name:    "invalid level between info and warn",
			at:      now,
			level:   Level(1),
			message: "message",
			wantErr: true,
		},
		{
			name:    "invalid level between warn and error",
			at:      now,
			level:   Level(5),
			message: "message",
			wantErr: true,
		},
		{
			name:    "invalid level between error and fatal",
			at:      now,
			level:   Level(9),
			message: "message",
			wantErr: true,
		},
		{
			name:    "invalid level above fatal",
			at:      now,
			level:   Level(13),
			message: "message",
			wantErr: true,
		},
		{
			name:    "empty field key",
			at:      now,
			level:   LevelInfo,
			message: "message",
			fields: []Field{
				stringField{
					value: "value",
				},
			},
			wantErr: true,
		},
		{
			name:    "one invalid field among valid fields",
			at:      now,
			level:   LevelInfo,
			message: "message",
			fields: []Field{
				stringField{
					key:   keys.New("first"),
					value: "one",
				},
				stringField{
					value: "invalid",
				},
				stringField{
					key:   keys.New("third"),
					value: "three",
				},
			},
			wantErr: true,
		},
		{
			name:    "multiple invalid values",
			at:      time.Time{},
			level:   Level(100),
			message: "",
			fields: []Field{
				stringField{
					value: "value",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := NewRecord(
				tt.at,
				tt.level,
				tt.message,
				tt.fields...,
			)

			err := record.Validate(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() expected error, got nil")
				}

				if !errors.Is(err, logerrors.ErrInvalidRecord) {
					t.Fatalf(
						"Validate() error = %v, want errors.Is(err, ErrInvalidRecord)",
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}

			if !record.Time.Equal(tt.at) {
				t.Errorf("Time = %v, want %v", record.Time, tt.at)
			}

			if record.Level != tt.level {
				t.Errorf("Level = %v, want %v", record.Level, tt.level)
			}

			if record.Message != tt.message {
				t.Errorf("Message = %q, want %q", record.Message, tt.message)
			}

			if len(record.Fields) != len(tt.fields) {
				t.Fatalf(
					"len(Fields) = %d, want %d",
					len(record.Fields),
					len(tt.fields),
				)
			}

			for i := range tt.fields {
				if record.Fields[i] != tt.fields[i] {
					t.Errorf(
						"Fields[%d] = %#v, want %#v",
						i,
						record.Fields[i],
						tt.fields[i],
					)
				}
			}
		})
	}
}

func TestNewRecord_CopiesFields(t *testing.T) {
	fields := []Field{
		stringField{
			key:   keys.New("key"),
			value: "original",
		},
	}

	record := NewRecord(
		time.Now(),
		LevelInfo,
		"message",
		fields...,
	)

	fields[0] = stringField{
		key:   keys.New("changed"),
		value: "changed",
	}

	if got, want := record.Fields[0].Key(), keys.New("key"); got != want {
		t.Errorf("Fields[0].Key = %q, want %q", got, want)
	}

	if got, want := record.Fields[0].String(), "original"; got != want {
		t.Errorf("Fields[0].String() = %q, want %q", got, want)
	}

	if err := record.Validate(context.Background()); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}
}
