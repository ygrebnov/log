package errors

import "github.com/ygrebnov/errorc"

var (
	ErrInvalidConfig            = errorc.New("invalid config")
	ErrCannotAccessLogFilePath  = errorc.New("cannot access log file path")
	ErrCannotResolveUserHomeDir = errorc.New("cannot resolve user home dir")
	ErrInvalidLogLevel          = errorc.New("invalid log level")
	ErrInvalidSinkKind          = errorc.New("invalid sink kind")
	ErrCannotOpenFileWriter     = errorc.New("cannot open file writer")
	ErrCannotFlushWriter        = errorc.New("cannot flush writer")
	ErrCannotWriteLogRecord     = errorc.New("cannot write log record")
)
