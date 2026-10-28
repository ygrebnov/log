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
	// Create a file logger that writes to the log file in the OS default application data directory.
    logger, err := log.NewFileLogger(log.WithAppName("myapp"))
    if err != nil {
        logpkg.Fatalf("cannot initialize logger: %v", err)
    }
    defer logger.Close()
    
    // Log a message with structured fields.
    logger.Warn("Disk space low", log.String("disk", "/dev/sda1"), log.Int("remaining_mb", 512))
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
                Format: log.FormatJSON,
                Level:  log.LevelDebug,
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

### File Sink Security

When no custom path is configured, the file sink creates <appName>.log in the operating system's default per-user application data directory. This is the recommended configuration for applications that need file-based logging.

A custom `path` may be configured when logs must be written to a specific location, such as a container volume or an application-managed log directory.

```go
import (
    logpkg "log"
	
    "github.com/ygrebnov/log"
)

func main() {
	// Relative paths are resolved against the process's current working directory, 
	// not the directory containing the executable.
    logger, err := log.NewFileLogger(log.WithPath("app.log"))
    if err != nil {
        logpkg.Fatalf("cannot initialize logger: %v", err)
    }
    defer logger.Close()
    logger.Error("application error", log.String("module", "auth"), log.String("user", "Bob"))
}
```

**Security considerations for custom paths:**

- Use a trusted directory whose path components cannot be replaced by untrusted users or processes.
- Avoid shared writable directories and attacker-controlled parent directories.
- Ensure that the application has the permissions required to create and append to the log file.
- Do not accept log paths directly from untrusted input.

The logger uses Go's `os.Root` to confine file operations to the opened parent directory. However, this does not prevent an attacker from redirecting the parent path before that directory is opened. Filesystem permissions and ownership remain the application's responsibility.

For containerized applications, prefer stdout/stderr logging where appropriate. Use a custom file path when persistent file-based logging is explicitly required.

## Installation

Requires Go 1.25.12 or later.

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