package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// ParseLevel converts a string representation to a slog.Level.
func ParseLevel(levelStr string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(levelStr)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New initializes a structured JSON logger writing to standard output.
func New(levelStr string) *slog.Logger {
	return NewWithWriter(os.Stdout, levelStr)
}

// NewWithWriter initializes a structured JSON logger writing to an arbitrary io.Writer (useful for testing).
func NewWithWriter(w io.Writer, levelStr string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: ParseLevel(levelStr),
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}
