package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/backend"
)

// TestWriteAPISuccess_MarshalError tests the error path when data cannot be marshaled.
// channels cannot be marshaled to JSON, so we use one as input.
func TestWriteAPISuccess_MarshalError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeAPISuccess(rec, make(chan int))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failure")
	}
}

// TestSessionManager_NoSessionManager tests that a non-SessionManager
// factory returns NoSessionManager.
func TestSessionManager_NonSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockFactory{name: "local command"},
	}
	sm := srv.sessionManager()
	if sm.IsAvailable() {
		t.Fatal("expected NoSessionManager to not be available")
	}
}

// TestHandleListSessions_GET_WithSessionManager tests the GET path with
// a mock session manager that returns data.
func TestHandleListSessions_GET_WithSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions", nil)
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
}

// TestHandleListSessions_POST_WithSessionManager tests the POST path.
func TestHandleListSessions_POST_WithSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	body := `{"name":"test-session"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// TestHandleListSessions_POST_InvalidName tests validation of session name.
func TestHandleListSessions_POST_InvalidName(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	body := `{"name":"../../../etc/passwd"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// TestHandleSessionDetail_GET_WithSessionManager tests GET session detail.
func TestHandleSessionDetail_GET_WithSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions/test-session", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// TestHandleSessionDetail_DELETE_WithSessionManager tests DELETE session.
func TestHandleSessionDetail_DELETE_WithSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/sessions/test-session", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// mockSessionManagerFactory is a mock factory that implements SessionManager.
type mockSessionManagerFactory struct{}

func (m *mockSessionManagerFactory) Name() string { return "mock-session" }
func (m *mockSessionManagerFactory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	return nil, nil
}

func (m *mockSessionManagerFactory) IsAvailable() bool { return true }
func (m *mockSessionManagerFactory) ListSessions() (json.RawMessage, error) {
	return json.RawMessage(`[{"name":"test"}]`), nil
}
func (m *mockSessionManagerFactory) GetSessionDetail(name string) (json.RawMessage, error) {
	return json.RawMessage(`{"name":"` + name + `"}`), nil
}
func (m *mockSessionManagerFactory) CreateSession(name string, command ...string) (string, error) {
	if name == "" {
		return "generated-session", nil
	}
	return name, nil
}
func (m *mockSessionManagerFactory) KillSession(name string) error { return nil }

// mockErrorSessionManager returns errors for session operations.
type mockErrorSessionManager struct{}

func (m *mockErrorSessionManager) Name() string { return "error-session" }
func (m *mockErrorSessionManager) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	return nil, nil
}
func (m *mockErrorSessionManager) IsAvailable() bool { return true }
func (m *mockErrorSessionManager) ListSessions() (json.RawMessage, error) {
	return nil, io.ErrUnexpectedEOF
}
func (m *mockErrorSessionManager) GetSessionDetail(name string) (json.RawMessage, error) {
	return nil, io.ErrUnexpectedEOF
}
func (m *mockErrorSessionManager) CreateSession(name string, command ...string) (string, error) {
	return "", io.ErrUnexpectedEOF
}
func (m *mockErrorSessionManager) KillSession(name string) error { return io.ErrUnexpectedEOF }

// Compile-time check that mocks satisfy the interfaces.
var _ backend.Factory = (*mockSessionManagerFactory)(nil)
var _ backend.SessionManager = (*mockSessionManagerFactory)(nil)
var _ backend.Factory = (*mockErrorSessionManager)(nil)
var _ backend.SessionManager = (*mockErrorSessionManager)(nil)

func TestHandleListSessions_GET_Error(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockErrorSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions", nil)
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListSessions_POST_EmptyName(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	body := `{}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListSessions_POST_CreateError(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockErrorSessionManager{},
	}
	body := `{"name":"test"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleSessionDetail_GET_Error(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockErrorSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/sessions/test-session", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSessionDetail_DELETE_Error(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockErrorSessionManager{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/sessions/test-session", nil)
	srv.handleSessionDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListSessions_POST_InvalidBody_WithSessionManager(t *testing.T) {
	srv := &Server{
		options: &Options{Path: "/"},
		factory: &mockSessionManagerFactory{},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/sessions", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleListSessions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
