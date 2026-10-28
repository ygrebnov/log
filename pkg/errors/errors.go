package errors

import "github.com/ygrebnov/errorc"

var (
	ErrInvalidConfig            = errorc.New("invalid config")
	ErrInvalidAppName           = errorc.New("invalid app name")
	ErrInsecurePermissions      = errorc.New("insecure permissions")
	ErrInvalidRecord            = errorc.New("invalid record")
	ErrCannotAccessLogFilePath  = errorc.New("cannot access log file path")
	ErrCannotResolveUserHomeDir = errorc.New("cannot resolve user home dir")
	ErrInvalidLogLevel          = errorc.New("invalid log level")
	ErrInvalidSinkKind          = errorc.New("invalid sink kind")
	ErrInvalidSinkFormat        = errorc.New("invalid sink format")
	ErrCannotOpenFileWriter     = errorc.New("cannot open file writer")
	ErrCannotFlushWriter        = errorc.New("cannot flush writer")
	ErrCannotWriteLogRecord     = errorc.New("cannot write log record")
)
