package logging

import (
	"log/slog"
	"os"

	"github.com/sumandas0/k8s-cluster-agent/internal/config"
)

// NewLogger creates a new structured logger based on configuration
func NewLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
	}

	switch cfg.LogFormat {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// parseLevel converts string log level to slog.Level
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
