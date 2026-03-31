package slogutil

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var updateGoldenSlog = flag.Bool("update", false, "update golden files")

// timeRe matches the JSON "time":"..." field in slog output.
var timeRe = regexp.MustCompile(`"time":"[^"]*"`)

func compareGoldenSlog(t *testing.T, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *updateGoldenSlog {
		t.Logf("updating golden file: %s", golden)
		_ = os.MkdirAll(filepath.Dir(golden), 0o755)
		_ = os.WriteFile(golden, got, 0o644)
	}
	want, err := os.ReadFile(golden) //nolint:gosec // reason: test code, golden file path is deterministic
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch:\n got: %s\nwant: %s", got, want)
	}
}

// stripTimestamps removes non-deterministic timestamps from slog JSON output.
func stripTimestamps(raw []byte) []byte {
	return timeRe.ReplaceAll(raw, []byte(`"time":"TIMESTAMP"`))
}

func TestSnapshot_New_DebugLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "debug")

	logger.Debug("debug msg", "key1", "val1")
	logger.Info("info msg", "key2", "val2")
	logger.Warn("warn msg")
	logger.Error("error msg")

	compareGoldenSlog(t, stripTimestamps(buf.Bytes()))
}

func TestSnapshot_New_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "warn")

	logger.Debug("should not appear")
	logger.Info("should not appear")
	logger.Warn("should appear", "severity", "high")
	logger.Error("also appears", "code", 500)

	compareGoldenSlog(t, stripTimestamps(buf.Bytes()))
}

func TestSnapshot_New_ErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "error")

	logger.Debug("no")
	logger.Info("no")
	logger.Warn("no")
	logger.Error("yes", "reason", "critical failure")

	compareGoldenSlog(t, stripTimestamps(buf.Bytes()))
}

func TestSnapshot_New_InvalidLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "nonexistent-level")

	logger.Debug("no")
	logger.Info("yes, defaults to info")
	logger.Warn("yes")
	logger.Error("yes")

	compareGoldenSlog(t, stripTimestamps(buf.Bytes()))
}

func TestSnapshot_New_MultipleKeyTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "debug")

	testErr := errors.New("something went wrong")
	logger.Info("structured fields",
		"string", "hello",
		"int", 42,
		"float", 3.14,
		"bool", true,
		"error", testErr,
	)

	compareGoldenSlog(t, stripTimestamps(buf.Bytes()))
}
