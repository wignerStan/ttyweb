package slogutil

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewJSONHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "debug")
	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("expected JSON level, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("expected key=value, got: %s", output)
	}
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("expected msg field, got: %s", output)
	}
}

func TestNewLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "warn")
	logger.Debug("should not appear")

	if buf.Len() > 0 {
		t.Errorf("debug should be suppressed at warn level, got: %s", buf.String())
	}

	logger.Warn("should appear")
	if !strings.Contains(buf.String(), `"level":"WARN"`) {
		t.Errorf("expected WARN level, got: %s", buf.String())
	}
}

func TestNewDefaultLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "invalid")
	logger.Info("appears at default info level")
	if !strings.Contains(buf.String(), `"level":"INFO"`) {
		t.Error("default should be info level")
	}
}
