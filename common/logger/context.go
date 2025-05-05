package logger

import (
	"context"
	"log/slog"
)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

// Logger context keys
const (
	LoggerKey ContextKey = "logger"
)

// FromContext retrieves the logger from the context
// If no logger is found, it returns the default logger
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// WithRequestID adds a request ID to the logger in the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx)
	loggerWithRequestID := logger.With("request_id", requestID)
	return WithLogger(ctx, loggerWithRequestID)
}
