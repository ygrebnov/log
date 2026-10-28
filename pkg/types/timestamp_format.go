package types

import "time"

type TimestampFormat string

const (
	TimestampFormatRFC3339Nano TimestampFormat = "rfc3339nano"
	TimestampFormatRFC3339     TimestampFormat = "rfc3339"
)

func (f TimestampFormat) Layout() string {
	switch f {
	case TimestampFormatRFC3339:
		return time.RFC3339
	default:
		return time.RFC3339Nano
	}
}
