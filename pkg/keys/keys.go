package keys

import "github.com/ygrebnov/keys"

var (
	log  = keys.Factory(keys.WithSegments("log"))
	sink = keys.Factory(keys.WithSegments("sink"))
)

var (
	LogFilePath = log("file.path")
	LogDir      = log("dir")
	LogLevel    = log("level")

	SinkKind   = sink("kind")
	SinkFormat = sink("format")

	Cause = keys.New("cause")
)
