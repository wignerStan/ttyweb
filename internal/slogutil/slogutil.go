// Package slogutil provides a factory for creating slog.Logger instances
// with JSON output and configurable log levels.
package slogutil

import (
	"io"
	"log/slog"
	"strings"
)

// New creates a *slog.Logger writing JSON to w.
// levelStr accepts "debug", "info", "warn", or "error".
// Invalid values default to "info".
func New(w io.Writer, levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
