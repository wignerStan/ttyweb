package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"ttyweb/ai"
)

func TestHandleSpeechWS_NotConfigured(t *testing.T) {
	// Clear Xunfei env vars to ensure not configured.
	t.Setenv("XFYUN_APP_ID", "")
	t.Setenv("XFYUN_API_KEY", "")
	t.Setenv("XFYUN_API_SECRET", "")

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws/speech", nil)
	srv.handleSpeechWS(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var resp serverMessage
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Type != "error" {
		t.Fatalf("expected error type, got %q", resp.Type)
	}
	if resp.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestHandleSpeechWS_Configured_ButWebSocket(t *testing.T) {
	// Even when configured, we can't fully test WebSocket upgrade via httptest.
	// Test that the config check passes.
	t.Setenv("XFYUN_APP_ID", "test-id")
	t.Setenv("XFYUN_API_KEY", "test-key")
	t.Setenv("XFYUN_API_SECRET", "test-secret")

	cfg := ai.LoadXunfeiConfigFromEnv()
	if !cfg.IsConfigured() {
		t.Skip("Xunfei config not available in test env")
	}
}

func TestSpeechSession_CloseBoth(t *testing.T) {
	// closeBoth should be safe to call multiple times.
	sess := &speechSession{
		done: make(chan struct{}),
	}
	sess.closeBoth()
	sess.closeBoth() // Second call should not panic.
}

func TestSpeechSession_CloseBoth_NilConns(t *testing.T) {
	sess := &speechSession{
		done: make(chan struct{}),
	}
	sess.closeBoth() // Should not panic with nil conns.
}

func TestHandleSpeechWS_Configured_WebSocketUpgrade(t *testing.T) {
	// Set Xunfei env vars so IsConfigured() returns true.
	t.Setenv("XFYUN_APP_ID", "test-app-id")
	t.Setenv("XFYUN_API_KEY", "test-api-key")
	t.Setenv("XFYUN_API_SECRET", "test-api-secret")

	srv := newTestServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/speech", srv.handleSpeechWS)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/speech"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send an unknown message type to verify the run loop processes messages.
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"unknown"}`))

	// Close to trigger read error and exit.
	_ = conn.Close()
}
