package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGolden_ListTaskAISessions_Empty covers the empty list path.
func TestGolden_ListTaskAISessions_Empty(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/kanban/tasks/task-123/sessions", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ListTaskAISessions_WithSessionID covers the method-not-allowed path
// when a session ID is provided to the list endpoint.
func TestGolden_ListTaskAISessions_WithSessionID(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/kanban/tasks/task-123/sessions/session-456", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_LinkTaskAISession_BadJSON covers invalid JSON on link.
func TestGolden_LinkTaskAISession_BadJSON(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/kanban/tasks/task-123/sessions",
		bytes.NewReader([]byte("not json")),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_LinkTaskAISession_WithSessionID covers the method-not-allowed path
// when a session ID is provided to the link (POST) endpoint.
func TestGolden_LinkTaskAISession_WithSessionID(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/kanban/tasks/task-123/sessions/session-456", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_UnlinkTaskAISession_NoSessionID covers the method-not-allowed path
// when no session ID is provided to the unlink (DELETE) endpoint.
func TestGolden_UnlinkTaskAISession_NoSessionID(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/kanban/tasks/task-123/sessions", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_UnlinkTaskAISession_NotFound covers unlink for nonexistent link.
func TestGolden_UnlinkTaskAISession_NotFound(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/kanban/tasks/task-123/sessions/nonexistent-session", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_LinkTaskAISession_Success covers successful session link.
func TestGolden_LinkTaskAISession_Success(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/kanban/tasks/task-123/sessions",
		bytes.NewReader([]byte(`{"ai_session_id":"session-abc"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Verify the response contains the expected fields.
	body := rec.Body.String()
	if body == "" || body[0] != '{' {
		t.Fatalf("expected JSON response, got: %s", body)
	}
}

// TestGolden_TaskAISession_InvalidPath covers invalid path format.
func TestGolden_TaskAISession_InvalidPath(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/kanban/tasks/invalid-structure", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_TaskAISession_MethodNotAllowed covers unsupported HTTP method.
func TestGolden_TaskAISession_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPatch,
		"/api/kanban/tasks/task-123/sessions", nil,
	)
	srv.handleTaskAISessionLinks(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_UnlinkTaskAISession_Success covers successful unlink.
func TestGolden_UnlinkTaskAISession_Success(t *testing.T) {
	srv := newTestServer()

	// First link a session so we can unlink it.
	linkRec := httptest.NewRecorder()
	linkReq := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/kanban/tasks/task-unlink/sessions",
		bytes.NewReader([]byte(`{"ai_session_id":"session-to-unlink"}`)),
	)
	linkReq.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(linkRec, linkReq)

	// Verify link succeeded.
	if linkRec.Code != http.StatusOK {
		t.Fatalf("link failed: %s", linkRec.Body.String())
	}

	// Now unlink it.
	unlinkRec := httptest.NewRecorder()
	unlinkReq := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/kanban/tasks/task-unlink/sessions/session-to-unlink", nil,
	)
	srv.handleTaskAISessionLinks(unlinkRec, unlinkReq)

	if unlinkRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", unlinkRec.Code, unlinkRec.Body.String())
	}
}
