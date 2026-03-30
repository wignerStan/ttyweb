package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTmuxConfig_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/config", nil)
	srv.handleTmuxConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data["label"] != "Ctrl+B" {
		t.Fatalf("expected 'Ctrl+B', got %v", resp.Data["label"])
	}
}

func TestHandleTmuxConfig_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/config", nil)
	srv.handleTmuxConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleQuickDirs_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/quick-dirs", nil)
	srv.handleQuickDirs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool  `json:"success"`
		Data    []any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Data == nil {
		t.Fatal("expected data to be non-nil array")
	}
	// Default config has no quick dirs, so expect empty array.
	if len(resp.Data) != 0 {
		t.Fatalf("expected empty dirs, got %d", len(resp.Data))
	}
}

func TestHandleQuickDirs_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/quick-dirs", nil)
	srv.handleQuickDirs(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTmuxNewWindow_NonTmuxBackend(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "local command"},
	}
	body := `{"session":"s1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-window", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewWindow(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleTmuxNewWindow_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/new-window", nil)
	srv.handleTmuxNewWindow(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTmuxNewSession_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"new-session"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-session", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewSession(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleTmuxNewSession_InvalidBody_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-session", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewSession(rec, req)

	// NoSessionManager returns 503 before parsing body.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleTmuxNewSession_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/new-session", nil)
	srv.handleTmuxNewSession(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTmuxTree_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success")
	}
	// NoSessionManager returns an empty array, not a sessions map.
}

func TestHandleTmuxTree_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTmuxSendKeys_NonTmuxBackend(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "local command"},
	}
	body := `{"pane":"%1","keys":"ls"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/send-keys", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxSendKeys(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleTmuxSendKeys_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/send-keys", nil)
	srv.handleTmuxSendKeys(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTmuxPaneMode_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/pane-mode?pane=pane1", nil)
	srv.handleTmuxPaneMode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data["mode"] != "pane" {
		t.Fatalf("expected 'pane', got %q", resp.Data["mode"])
	}
}

func TestHandleTmuxPaneMode_POST(t *testing.T) {
	srv := newTestServer()
	body := `{"mode":"control"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/pane-mode?pane=pane1", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxPaneMode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data["mode"] != "control" {
		t.Fatalf("expected 'control', got %q", resp.Data["mode"])
	}

	// Verify the mode persists via GET.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/pane-mode?pane=pane1", nil)
	srv.handleTmuxPaneMode(rec2, req2)

	var resp2 struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp2.Data["mode"] != "control" {
		t.Fatalf("expected persisted 'control', got %q", resp2.Data["mode"])
	}
}

func TestHandleTmuxPaneMode_MissingPane(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/pane-mode", nil)
	srv.handleTmuxPaneMode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxPaneMode_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/pane-mode?pane=pane1", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxPaneMode(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxPaneMode_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/tmux/pane-mode?pane=pane1", nil)
	srv.handleTmuxPaneMode(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTmuxNewWindow_InvalidBody(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-window", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewWindow(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxNewWindow_MissingSession(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	body := `{"name":"w1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-window", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewWindow(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxNewWindow_WithTmuxBackend(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	body := `{"session":"nonexistent-session","name":"test-win"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-window", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewWindow(rec, req)

	// tmux binary may not be available, so we accept 500 or success.
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
		t.Fatalf("expected 500 or 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTmuxSendKeys_InvalidBody(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/send-keys", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxSendKeys(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxSendKeys_MissingPane(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	body := `{"keys":"ls"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/send-keys", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxSendKeys(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxSendKeys_MissingKeys(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	body := `{"pane":"%1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/send-keys", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxSendKeys(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTmuxSendKeys_WithTmuxBackend(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	body := `{"pane":"%9999","keys":"ls -la"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/send-keys", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxSendKeys(rec, req)

	// tmux binary may not be available, so we accept 500 or success.
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
		t.Fatalf("expected 500 or 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTmuxNewSession_EmptyBody(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	body := `{}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-session", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTmuxNewSession_CreateError(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockErrorSessionManager{},
	}
	body := `{"name":"fail-session"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-session", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewSession(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTmuxNewSession_InvalidBody(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-session", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewSession(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTmuxNewWindow_WithDir(t *testing.T) {
	srv := &Server{
		options: &Options{},
		factory: &mockFactory{name: "tmux"},
	}
	body := `{"session":"s1","name":"w1","dir":"/tmp"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tmux/new-window", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTmuxNewWindow(rec, req)

	// tmux may not be available.
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
		t.Fatalf("expected 500 or 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
