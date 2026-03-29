package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/backend"
)

func TestHandleListSessions_GET_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions", nil)
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
}

func TestHandleListSessions_POST_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"test-session"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleListSessions_POST_InvalidBody_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	// NoSessionManager returns 503 before parsing body.
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleListSessions_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/sessions", nil)
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSessionDetail_GET_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions/test", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleSessionDetail_DELETE_NoSessionManager(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/sessions/test", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleSessionDetail_EmptyName(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions/", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSessionDetail_InvalidName(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions/../../etc/passwd", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSessionDetail_MethodNotAllowed_NoSessionManager(t *testing.T) {
	// NoSessionManager returns 503 before checking method.
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions/test", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleListBackends_LocalBackend(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockFactory{name: "local command"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/backends", nil)
	srv.handleListBackends(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 backends, got %d", len(resp.Data))
	}
}

func TestHandleListBackends_TmuxBackend(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockFactory{name: "tmux"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/backends", nil)
	srv.handleListBackends(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSessionManager_NoSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockFactory{name: "local command"},
	}
	sm := srv.sessionManager()
	if sm.IsAvailable() {
		t.Fatal("expected NoSessionManager to not be available")
	}
	if _, ok := sm.(backend.NoSessionManager); !ok {
		t.Fatal("expected NoSessionManager type")
	}
}

// mockFactory is a minimal mock for testing.
type mockFactory struct {
	name string
}

func (m *mockFactory) Name() string { return m.name }
func (m *mockFactory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	return nil, nil
}

func TestHandleSessionDetail_PUT_MethodNotAllowed(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/sessions/test", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleListBackends_POST_ReturnsBackends(t *testing.T) {
	// handleListBackends doesn't check method — it always returns the list.
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockFactory{name: "local command"},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/backends", nil)
	srv.handleListBackends(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
