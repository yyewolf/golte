package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup configures the global logger based on the provided configuration
func Setup(level, format string) error {
	// Parse log level
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Set the global logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

// WithFields returns a logger with the given fields
func WithFields(fields ...any) *slog.Logger {
	return slog.With(fields...)
}

// WithComponent returns a logger with a component field
func WithComponent(component string) *slog.Logger {
	return slog.With("component", component)
}
