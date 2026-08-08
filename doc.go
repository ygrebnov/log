// Package log provides structured, asynchronous logging.
//
// A Logger dispatches log records to one or more configured sinks.
// Each sink owns its own queue, buffering and output, allowing different
// destinations (for example stderr and files) to use independent log levels
// and formatting.
//
// Configuration is represented by Config. Call ApplyDefaults before Validate
// when using partially specified configurations.
//
// Example:
//
//	logger, err := log.NewStdoutLogger()
//	if err != nil {
//		log.Fatalf("cannot initialize logger: %v", err)
//	}
//	defer logger.Close()
//
//	logger.Info("server started", log.String(keys.Address, ":8080"))
package log
