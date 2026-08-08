package keys

import "github.com/ygrebnov/keys"

var (
	log = keys.Factory(keys.WithSegments("log"))
)

var (
	LogFilePath = log("file.path")
	LogDir      = log("dir")
	LogLevel    = log("level")

	SinkKind = keys.New("sink.kind")

	Cause = keys.New("cause")
)
