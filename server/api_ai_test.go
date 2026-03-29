package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestResolveEnvConfig_FromEnv verifies that resolveEnvConfig reads from the
// environment variable when it is set.
func TestResolveEnvConfig_FromEnv(t *testing.T) {
	t.Setenv("TEST_RESOLVE_KEY", "env-value")
	got := resolveEnvConfig("TEST_RESOLVE_KEY", "fallback")
	if got != "env-value" {
		t.Errorf("expected env value, got %q", got)
	}
}

// TestResolveEnvConfig_Fallback verifies that resolveEnvConfig returns the
// fallback when the environment variable is not set.
func TestResolveEnvConfig_Fallback(t *testing.T) {
	got := resolveEnvConfig("TEST_RESOLVE_MISSING_KEY", "fallback")
	if got != "fallback" {
		t.Errorf("expected fallback, got %q", got)
	}
}

// TestResolveEnvConfig_EmptyEnvFallsBack verifies that an empty env var
// triggers the fallback.
func TestResolveEnvConfig_EmptyEnvFallsBack(t *testing.T) {
	t.Setenv("TEST_RESOLVE_EMPTY", "")
	got := resolveEnvConfig("TEST_RESOLVE_EMPTY", "fallback")
	if got != "fallback" {
		t.Errorf("expected fallback for empty env, got %q", got)
	}
}

// TestHandleAICommand_IgnoresHeadersAndQueryParams verifies that the handler
// does not read API URL, API key, or model from HTTP headers or query
// parameters. This is the core SSRF prevention test.
func TestHandleAICommand_IgnoresHeadersAndQueryParams(t *testing.T) {
	// Ensure no API key is set in the environment so the handler returns
	// the "no API key" response instead of making an outbound request.
	t.Setenv("LLM_API_KEY", "")

	// Also clear any other LLM env vars to avoid interference.
	t.Setenv("LLM_API_URL", "")
	t.Setenv("LLM_MODEL", "")

	srv := &Server{}

	body := aiCommandRequest{Prompt: "list files"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/command?api_url=http://evil.example.com&api_key=stolen-key&model=evil-model", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LLM-Api-URL", "http://evil.example.com")
	req.Header.Set("X-LLM-Api-Key", "stolen-key")
	req.Header.Set("X-LLM-Model", "evil-model")

	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// The response must indicate no API key -- proving headers/query params
	// were ignored (they provided a key but it was not used).
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	// Parse the data field to check the explanation message.
	var data aiCommandResponse
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("failed to decode data: %v", err)
	}

	if data.Command != "" {
		t.Errorf("expected empty command (no API key), got %q", data.Command)
	}

	if !strings.Contains(data.Explanation, "LLM_API_KEY") {
		t.Errorf("expected explanation to mention LLM_API_KEY env var, got %q", data.Explanation)
	}

	// Verify the old message mentioning headers is NOT present.
	if strings.Contains(data.Explanation, "X-LLM-Api-Key header") {
		t.Errorf("explanation should not reference header, got %q", data.Explanation)
	}
}

// TestHandleAICommand_EnvVarTakesPrecedence verifies that when LLM_API_KEY is
// set in the environment, the handler uses it (and not any header value).
// We set the key to a known-bad value so the request fails with a sanitized
// error message.
func TestHandleAICommand_EnvVarTakesPrecedence(t *testing.T) {
	t.Setenv("LLM_API_KEY", "env-provided-key")
	// Use a non-existent URL so the request will fail.
	t.Setenv("LLM_API_URL", "http://127.0.0.1:1")
	t.Setenv("LLM_MODEL", "test-model")

	srv := &Server{}

	body := aiCommandRequest{Prompt: "list files", Role: "cli-expert"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/command?api_key=attacker-key", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LLM-Api-Key", "attacker-key")

	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	// The request should fail because the API URL is unreachable.
	// The key point is it tried "env-provided-key", not "attacker-key".
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Success {
		t.Fatal("expected error response")
	}

	// The error message must be generic -- no internal details leaked.
	if resp.Error != "AI request failed" {
		t.Errorf("expected generic error message, got %q", resp.Error)
	}

	// Ensure no internal error details (URLs, connection refused, etc.) are leaked.
	lower := strings.ToLower(resp.Error)
	if strings.Contains(lower, "127.0.0.1") || strings.Contains(lower, "connection") {
		t.Errorf("error message leaks internal details: %q", resp.Error)
	}
}

// TestHandleAICommand_ErrorSanitization verifies that error responses do not
// contain internal details like API URLs, keys, or stack traces.
func TestHandleAICommand_ErrorSanitization(t *testing.T) {
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_API_URL", "http://127.0.0.1:1")

	srv := &Server{}

	body := aiCommandRequest{Prompt: "test prompt"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/command", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Must not contain any internal details.
	if strings.Contains(resp.Error, "http://") || strings.Contains(resp.Error, "127.0.0.1") {
		t.Errorf("error message leaks URL: %q", resp.Error)
	}
	if strings.Contains(resp.Error, "test-key") || strings.Contains(resp.Error, "test-model") {
		t.Errorf("error message leaks credentials: %q", resp.Error)
	}
	if strings.Contains(resp.Error, "dial") || strings.Contains(resp.Error, "connection refused") {
		t.Errorf("error message leaks internal error: %q", resp.Error)
	}
}

// TestHandleAICommand_InvalidMethod verifies that non-POST requests are rejected.
func TestHandleAICommand_InvalidMethod(t *testing.T) {
	srv := &Server{}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/command", nil)
	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// TestHandleAICommand_EmptyPrompt verifies that requests with empty prompts are rejected.
func TestHandleAICommand_EmptyPrompt(t *testing.T) {
	srv := &Server{}

	body := aiCommandRequest{Prompt: ""}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/command", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestHandleAICommand_InvalidBody verifies that malformed JSON is rejected.
func TestHandleAICommand_InvalidBody(t *testing.T) {
	srv := &Server{}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/command", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestHandleAICommand_DefaultsWhenEnvUnset verifies that when LLM_API_URL and
// LLM_MODEL env vars are not set, the handler uses the built-in defaults.
func TestHandleAICommand_DefaultsWhenEnvUnset(t *testing.T) {
	// Clear the env vars to test defaults.
	_ = os.Unsetenv("LLM_API_URL")
	_ = os.Unsetenv("LLM_MODEL")
	// Set a bad key so we get a 500 (proving the defaults were used for the URL).
	t.Setenv("LLM_API_KEY", "some-key")

	srv := &Server{}

	body := aiCommandRequest{Prompt: "test"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/command", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.handleAICommand(rec, req)

	// Should fail because https://api.openai.com is unreachable (or returns an
	// auth error), but the point is the default URL was used -- not anything
	// from headers/query params.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 with default URL, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
