**log** is a type-safe, asynchronous logging library for Go supporting multiple output sinks and structured fields. Written by [ygrebnov](https://github.com/ygrebnov).

---

[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/log)](https://pkg.go.dev/github.com/ygrebnov/log)
[![Build Status](https://github.com/ygrebnov/log/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/log/actions/workflows/build.yml)

## Features
- Structured logging with typed fields
- Asynchronous log writing with buffered records. Call Logger.Close() before program exit to flush pending records.
- Multiple output sinks (stdout, stderr, file)
- Configurable log levels (Trace, Debug, Info, Warn, Error, Fatal)
- Serializable configuration with ApplyDefaults() and Validate()

## Usage
### stdout logger
```go
import (
    logpkg "log"

    "github.com/ygrebnov/log"
)

func main() {
    logger, err := log.NewStdoutLogger()
    if err != nil {
        logpkg.Fatalf("cannot initialize logger: %v", err)
    }
    defer logger.Close()
	
	// Log a message with structured fields.
    logger.Info("server started", log.String("host", "localhost"), log.Int("port", 8080))
	
	// Use Log() to log with a custom level.
    logger.Log(log.LevelDebug, "request received", log.Bool("debug", true))
}
```

### File logger
```go
import (
    logpkg "log"
	
    "github.com/ygrebnov/log"
)

func main() {
    logger, err := log.NewFileLogger(log.WithPath("app.log"))
    if err != nil {
        logpkg.Fatalf("cannot initialize logger: %v", err)
    }
    defer logger.Close()
    
    // Log a message with structured fields.
    logger.Warn("Disk space low", log.String("disk", "/dev/sda1"), log.Int("remaining_mb", 512))
	
	// Alternatively, provide an application name to store the log file in the operating system’s default application data directory.
    logger2, err := log.NewFileLogger(log.WithAppName("myapp"))
    if err != nil {
        logpkg.Fatalf("cannot initialize logger: %v", err)
    }
    defer logger2.Close()
    logger2.Error("application error", log.String("module", "auth"), log.String("user", "Bob"))
}
```

### Configurable multi-sink logger
```go
import (
	logpkg "log"
	
    "github.com/ygrebnov/log"
)

func main() {
	// Create a logger configuration with multiple sinks (stdout and file).
    cfg := &log.Config{
        AppName: "myapp",
        Sinks: []log.SinkConfig{
            {
                Kind:   log.KindStdOut,
                Format: log.FormatText,
                Level:  log.LevelInfo,
            },
            {
                Kind:   log.KindFile,
                Format: log.FormatText,
                Level:  log.LevelDebug,
                Path:   "app.log",
            },
        },
    }
	
	// Apply default values to any fields you did not specify.
    if err := cfg.ApplyDefaults(); err != nil {
        logpkg.Fatalf("cannot apply log default configuration: %v", err)
    }

	// Create a logger based on the configuration.
    logger, err := log.NewLogger(cfg)
    if err != nil {
        logpkg.Fatalf("cannot initialize logger: %v", err)
    }
    defer logger.Close()
    
    // Log messages will be sent to both stdout and the file.
    logger.Info("application started", log.String("version", "1.0.0"))
}
```

## Installation

Requires Go 1.22 or later.

```shell
go get github.com/ygrebnov/log@latest
```

## Versioning
This library is pre-1.0. Before v1.0.0, breaking changes may occur in minor releases. Once it reaches 1.0, semantic versioning will apply more strictly.

## Contributing

Contributions are welcome!  
Please open an [issue](https://github.com/ygrebnov/log/issues) or submit a [pull request](https://github.com/ygrebnov/log/pulls).

## License

Distributed under the BSD 3-Clause License. See the [LICENSE](LICENSE) file for details.