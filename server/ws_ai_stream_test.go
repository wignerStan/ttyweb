package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ttyweb/db"
)

func TestSanitizeStreamError_AlwaysReturnsGeneric(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"URLs", fmt.Errorf("API returned status 500: https://internal.company.com/v1/chat/completions failed")},
		{"Bearer tokens", fmt.Errorf("Authorization failed: Bearer sk-12345-secret-key")},
		{"Long error", fmt.Errorf("error: %s", strings.Repeat("a", 200))},
		{"Short safe error", fmt.Errorf("context canceled")},
		{"SK prefix", fmt.Errorf("invalid api key sk-proj-abc123")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := sanitizeStreamError(tt.err)
			if msg != "AI stream error" {
				t.Errorf("expected generic error, got %q", msg)
			}
		})
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

func TestHandleAIStream_InvalidJSON(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .server.Version }}"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create an HTTP test server that serves the WebSocket handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/ai/stream", srv.handleAIStream)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/ai/stream"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = ws.Close() }()

	// Send invalid JSON.
	err = ws.WriteMessage(websocket.TextMessage, []byte("not json"))
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Should receive an error message.
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var streamMsg wsStreamMessage
	if err := json.Unmarshal(msg, &streamMsg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if streamMsg.Type != "error" {
		t.Errorf("expected error type, got %q", streamMsg.Type)
	}
}

func TestHandleAIStream_AuthFailed(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .server.Version }}"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/ai/stream", srv.handleAIStream)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/ai/stream"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = ws.Close() }()

	// Send request with wrong auth token.
	reqBody := map[string]string{
		"role":       "cli",
		"prompt":     "hello",
		"auth_token": "wrong-token",
	}
	data, _ := json.Marshal(reqBody)
	err = ws.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var streamMsg wsStreamMessage
	if err := json.Unmarshal(msg, &streamMsg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !strings.Contains(streamMsg.Data, "authentication") {
		t.Errorf("expected authentication error, got %q", streamMsg.Data)
	}
}

func TestHandleAIStream_MissingPrompt(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .server.Version }}"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/ai/stream", srv.handleAIStream)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/ai/stream"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = ws.Close() }()

	// Send request without prompt.
	reqBody := map[string]string{
		"role": "cli",
	}
	data, _ := json.Marshal(reqBody)
	err = ws.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var streamMsg wsStreamMessage
	if err := json.Unmarshal(msg, &streamMsg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !strings.Contains(streamMsg.Data, "prompt") {
		t.Errorf("expected prompt error, got %q", streamMsg.Data)
	}
}

func TestResolveSystemPrompt_UnknownRole(t *testing.T) {
	prompt := resolveSystemPrompt("nonexistent-role")
	if prompt != defaultSystemPrompt {
		t.Errorf("expected default prompt for unknown role, got %q", prompt)
	}
}

func TestResolveSystemPrompt_KnownRole(t *testing.T) {
	prompt := resolveSystemPrompt("cli")
	// The cli role maps to store ID 1, which is a builtin role.
	if prompt == "" {
		t.Error("expected non-empty prompt for known role")
	}
	if prompt == defaultSystemPrompt {
		t.Error("expected role-specific prompt, not default")
	}
}
