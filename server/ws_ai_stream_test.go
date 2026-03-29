package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestSanitizeStreamError_RedactsURLs(t *testing.T) {
	err := fmt.Errorf("API returned status 500: https://internal.company.com/v1/chat/completions failed")
	msg := sanitizeStreamError(err)

	if msg != "upstream AI service error" {
		t.Errorf("expected redacted URL error, got %q", msg)
	}
}

func TestSanitizeStreamError_RedactsBearerTokens(t *testing.T) {
	err := fmt.Errorf("Authorization failed: Bearer sk-12345-secret-key")
	msg := sanitizeStreamError(err)

	if msg != "authentication error with AI service" {
		t.Errorf("expected redacted auth error, got %q", msg)
	}
}

func TestSanitizeStreamError_TruncatesLongErrors(t *testing.T) {
	longMsg := strings.Repeat("a", 200)
	err := fmt.Errorf("error: %s", longMsg)
	msg := sanitizeStreamError(err)

	if msg != "upstream AI service error" {
		t.Errorf("expected truncated error, got %q", msg)
	}
}

func TestSanitizeStreamError_PreservesShortSafeErrors(t *testing.T) {
	err := fmt.Errorf("context canceled")
	msg := sanitizeStreamError(err)

	expected := "AI stream error: context canceled"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestSanitizeStreamError_RedactsSKPrefix(t *testing.T) {
	err := fmt.Errorf("invalid api key sk-proj-abc123")
	msg := sanitizeStreamError(err)

	if msg != "authentication error with AI service" {
		t.Errorf("expected redacted sk- error, got %q", msg)
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_ENV_VAR_12345", "hello")
	got := envOrDefault("TEST_ENV_VAR_12345", "default")
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}

	got = envOrDefault("NONEXISTENT_VAR_XYZ", "fallback")
	if got != "fallback" {
		t.Errorf("expected %q, got %q", "fallback", got)
	}
}
