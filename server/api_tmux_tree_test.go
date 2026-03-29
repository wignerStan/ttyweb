package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/backend"
)

// mockTreeSessionManager returns pre-canned session list and detail data
// for testing the tmux tree handler.
type mockTreeSessionManager struct{}

func (_ *mockTreeSessionManager) Name() string { return "mock-tree" }
func (_ *mockTreeSessionManager) New(map[string][]string, map[string][]string) (backend.Slave, error) {
	return nil, nil
}
func (_ *mockTreeSessionManager) IsAvailable() bool { return true }
func (_ *mockTreeSessionManager) ListSessions() (json.RawMessage, error) {
	return json.RawMessage(`[{"name":"session1"},{"name":"session2"}]`), nil
}
func (_ *mockTreeSessionManager) GetSessionDetail(name string) (json.RawMessage, error) {
	switch name {
	case "session1":
		return json.RawMessage(`{"panes":[{"id":"%0","window":0,"title":"bash","current_command":"vim"},{"id":"%1","window":0,"title":"bash","current_command":"ls"}]}`), nil
	case "session2":
		return json.RawMessage(`{"panes":[{"id":"%2","window":1,"title":"bash","current_command":"top"}]}`), nil
	default:
		return nil, io.ErrUnexpectedEOF
	}
}
func (_ *mockTreeSessionManager) CreateSession(string, ...string) (string, error) {
	return "created", nil
}
func (_ *mockTreeSessionManager) KillSession(string) error { return nil }

var _ backend.Factory = (*mockTreeSessionManager)(nil)
var _ backend.SessionManager = (*mockTreeSessionManager)(nil)

func TestHandleTmuxTree_WithSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockTreeSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success")
	}

	sessions, ok := resp.Data["sessions"].([]any)
	if !ok {
		t.Fatal("expected sessions array")
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestHandleTmuxTree_ListError(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockErrorSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// mockInvalidJSONSessionManager returns unparseable JSON for sessions.
type mockInvalidJSONSessionManager struct{}

func (_ *mockInvalidJSONSessionManager) Name() string { return "mock-invalid-json" }
func (_ *mockInvalidJSONSessionManager) New(map[string][]string, map[string][]string) (backend.Slave, error) {
	return nil, nil
}
func (_ *mockInvalidJSONSessionManager) IsAvailable() bool { return true }
func (_ *mockInvalidJSONSessionManager) ListSessions() (json.RawMessage, error) {
	return json.RawMessage(`[{"name":"s1"},{"name":"s2"}`), nil
}
func (_ *mockInvalidJSONSessionManager) GetSessionDetail(string) (json.RawMessage, error) {
	return json.RawMessage(`{"panes":[]}`), nil
}
func (_ *mockInvalidJSONSessionManager) CreateSession(string, ...string) (string, error) {
	return "", nil
}
func (_ *mockInvalidJSONSessionManager) KillSession(string) error { return nil }

var _ backend.Factory = (*mockInvalidJSONSessionManager)(nil)
var _ backend.SessionManager = (*mockInvalidJSONSessionManager)(nil)

func TestHandleTmuxTree_InvalidSessionJSON(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockInvalidJSONSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

type mockListEmptySessionsManager struct{}

func (_ *mockListEmptySessionsManager) Name() string { return "mock-empty" }
func (_ *mockListEmptySessionsManager) New(map[string][]string, map[string][]string) (backend.Slave, error) {
	return nil, nil
}
func (_ *mockListEmptySessionsManager) IsAvailable() bool { return true }
func (_ *mockListEmptySessionsManager) ListSessions() (json.RawMessage, error) {
	return json.RawMessage(`[]`), nil
}
func (_ *mockListEmptySessionsManager) GetSessionDetail(string) (json.RawMessage, error) {
	return json.RawMessage(`{"panes":[]}`), nil
}
func (_ *mockListEmptySessionsManager) CreateSession(string, ...string) (string, error) {
	return "", nil
}
func (_ *mockListEmptySessionsManager) KillSession(string) error { return nil }

var _ backend.Factory = (*mockListEmptySessionsManager)(nil)
var _ backend.SessionManager = (*mockListEmptySessionsManager)(nil)

func TestHandleTmuxTree_EmptySessions(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockListEmptySessionsManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// mockDetailErrorSessionManager returns error for GetSessionDetail.
type mockDetailErrorSessionManager struct {
	mockListEmptySessionsManager
}

func (_ *mockDetailErrorSessionManager) GetSessionDetail(string) (json.RawMessage, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestHandleTmuxTree_DetailError(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockDetailErrorSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// mockInvalidDetailJSONSessionManager returns unparseable detail JSON.
type mockInvalidDetailJSONSessionManager struct {
	mockListEmptySessionsManager
}

func (_ *mockInvalidDetailJSONSessionManager) GetSessionDetail(string) (json.RawMessage, error) {
	return json.RawMessage(`{"panes":[{`), nil
}

func TestHandleTmuxTree_InvalidDetailJSON(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockInvalidDetailJSONSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tmux/tree", nil)
	srv.handleTmuxTree(rec, req)

	// Invalid detail JSON is skipped (continue), returns empty tree.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
