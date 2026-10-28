package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"opspilot/task-api/internal/platform/logger"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"UNKNOWN", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		got := logger.ParseLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer
	log := logger.NewWithWriter(&buf, "INFO")

	log.Info("test message", "key", "value")

	var output map[string]any
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON log output: %v, raw: %s", err, buf.String())
	}

	if output["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got %v", output["msg"])
	}
	if output["key"] != "value" {
		t.Errorf("expected key 'value', got %v", output["key"])
	}
	if output["level"] != "INFO" {
		t.Errorf("expected level 'INFO', got %v", output["level"])
	}
}
