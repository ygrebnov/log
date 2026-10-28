package core

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/ygrebnov/keys"
	"github.com/ygrebnov/log/pkg/types"
)

type textFormatter struct {
	timestampFormat types.TimestampFormat
}

func (f textFormatter) Format(r types.Record) ([]byte, error) {
	var b strings.Builder

	b.WriteString(r.Time.Format(f.timestampFormat.Layout()))
	b.WriteByte(' ')
	b.WriteString(strings.ToUpper(r.Level.String()))
	b.WriteByte(' ')
	b.WriteString(escapeTextMessage(r.Message))

	hasFields := false

	for _, field := range r.Fields {
		if field == nil {
			continue
		}

		if !hasFields {
			b.WriteByte(',')
			hasFields = true
		}

		b.WriteByte(' ')
		b.WriteString(escapeTextKey(field.Key()))
		b.WriteByte('=')
		b.WriteString(formatTextFieldValue(field))
	}

	b.WriteByte('\n')

	return []byte(b.String()), nil
}

// formatTextFieldValue quotes textual fields while leaving numeric and boolean
// fields unquoted. A nil error is represented as null.
func formatTextFieldValue(field types.Field) string {
	switch field.Kind() {
	case types.FieldKindNull:
		return "null"

	case types.FieldKindInt,
		types.FieldKindInt64,
		types.FieldKindBool:
		return field.String()

	case types.FieldKindString:
		return strconv.Quote(field.String())

	default:
		return strconv.Quote(field.String())
	}
}

// escapeTextKey quotes keys containing separators or control characters.
func escapeTextKey(key keys.Key) string {
	s := string(key)

	if strings.IndexFunc(s, func(r rune) bool {
		return r <= ' ' ||
			r == 0x7f ||
			r == '=' ||
			r == '"' ||
			r == '\\' ||
			!unicode.IsPrint(r)
	}) == -1 {
		return s
	}

	return strconv.Quote(s)
}

// escapeTextMessage preserves ordinary messages without quotes while escaping
// line breaks and other control characters so one record occupies one line.
func escapeTextMessage(message string) string {
	if strings.IndexFunc(message, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t' || r < 0x20 || r == 0x7f
	}) == -1 {
		return message
	}

	quoted := strconv.Quote(message)
	return quoted[1 : len(quoted)-1]
}
