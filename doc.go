// Package log provides structured, asynchronous logging.
//
// The logger prioritizes application availability over log delivery.
// Logging is best-effort: records may be dropped when a sink's queue is full,
// when formatting fails, or when the sink encounters an output error.
//
// Each sink has its own queue, buffer, output destination, log level, and
// formatter. A failure in one sink does not prevent other sinks from operating.
//
// Close attempts to process all records accepted into each sink's queue,
// flush buffered output, and release resources. It does not guarantee
// that every logging call is persisted.
//
// Configuration is represented by Config. Call ApplyDefaults before Validate
// when using partially specified configurations.
//
// Example:
//
//	logger, err := log.NewStdoutLogger()
//	if err != nil {
//		stdliblog.Fatalf("cannot initialize logger: %v", err)
//	}
//	defer logger.Close()
//
//	logger.Info("server started", log.String(keys.Address, ":8080"))
package log
