package logging

import (
	"context"
	"log/slog"
)

type contextKey string

const loggerKey contextKey = "logger"

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context, returns a default logger if not found
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithRequestID adds a request ID to the logger in context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := FromContext(ctx).With("request_id", requestID)
	return WithLogger(ctx, logger)
}
