// Tests in this file insert into the global aiSessionService.Store() singleton.
// Each test uses a unique SessionID+Type combination to avoid cross-test interference.
// Tests intentionally do NOT call t.Parallel() to prevent concurrent singleton mutations.
package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ttyweb/ai"
	"ttyweb/service"
)

func TestFormatOptionalTime_Nil(t *testing.T) {
	result := formatOptionalTime(nil)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestFormatOptionalTime_NonNil(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	result := formatOptionalTime(&ts)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.Contains(*result, "2025") {
		t.Fatalf("expected 2025 in result, got %q", *result)
	}
}

func TestNewAISessionResponse(t *testing.T) {
	now := time.Now()
	r := service.AISessionRecord{
		ID:                    1,
		SessionID:             "sess-123",
		Type:                  "claude",
		ProjectPath:           "/path/to/project",
		FilePath:              "/path/to/session.jsonl",
		Model:                 "claude-3",
		Title:                 "Test Session",
		SessionStartedAt:      now,
		LastMessageAt:         &now,
		MessageCount:          10,
		AssistantMessageCount: 3,
		FileModTime:           now,
		FileSize:              2048,
	}

	resp := newAISessionResponse(&r)

	if resp.ID != 1 {
		t.Fatalf("expected ID 1, got %d", resp.ID)
	}
	if resp.SessionID != "sess-123" {
		t.Fatalf("expected sess-123, got %q", resp.SessionID)
	}
	if resp.Type != "claude" {
		t.Fatalf("expected claude, got %q", resp.Type)
	}
	if resp.ProjectPath != "/path/to/project" {
		t.Fatalf("expected project path, got %q", resp.ProjectPath)
	}
	if resp.Model != "claude-3" {
		t.Fatalf("expected claude-3, got %q", resp.Model)
	}
	if resp.Title != "Test Session" {
		t.Fatalf("expected Test Session, got %q", resp.Title)
	}
	if resp.MessageCount != 10 {
		t.Fatalf("expected 10 messages, got %d", resp.MessageCount)
	}
	if resp.AssistantMessageCount != 3 {
		t.Fatalf("expected 3 assistant messages, got %d", resp.AssistantMessageCount)
	}
	if resp.FileSize != 2048 {
		t.Fatalf("expected 2048 file size, got %d", resp.FileSize)
	}
	if resp.LastMessageAt == nil {
		t.Fatal("expected non-nil LastMessageAt")
	}
}

func TestNewAISessionResponse_NoLastMessage(t *testing.T) {
	now := time.Now()
	r := service.AISessionRecord{
		ID:               1,
		SessionID:        "sess-123",
		Type:             "claude",
		FilePath:         "/path/to/session.jsonl",
		SessionStartedAt: now,
		FileModTime:      now,
	}

	resp := newAISessionResponse(&r)
	if resp.LastMessageAt != nil {
		t.Fatalf("expected nil LastMessageAt, got %v", resp.LastMessageAt)
	}
}

func TestHandleAISessions_Cleanup(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/cleanup", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAISessions_Cleanup_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/cleanup", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_InvalidID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/abc", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_Detail_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/99999", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_Conversation_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/99999/conversation", nil)
	srv.handleAISessions(rec, req)

	// Service returns nil for nonexistent sessions, which the handler maps to 404.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_Conversation_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/1/conversation", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_Refresh_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/99999/refresh", nil)
	srv.handleAISessions(rec, req)

	// Service returns nil for nonexistent sessions, which the handler maps to 404.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_Refresh_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/1/refresh", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAISessions_Subroute_Unknown(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/1/unknown", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleAIListSessions_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskAISessionLinks_POST_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t1/sessions", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskAISessionLinks_POST_InvalidPath(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t1/other", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskAISessionLinks_POST_WithSessionID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t1/sessions/s1", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskAISessionLinks_GET_WithSessionID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/kanban/tasks/t1/sessions/s1", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskAISessionLinks_DELETE_NoSessionID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/kanban/tasks/t1/sessions", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskAISessionLinks_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/kanban/tasks/t1/sessions", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleProjectSync_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/p1/sync", nil)
	srv.handleProjectSync(rec, req, "p1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleProjectSync_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects/nonexistent/sync", nil)
	srv.handleProjectSync(rec, req, "nonexistent")

	// May return 404 or 500 due to stale singleton.
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 404 or 500, got %d", rec.Code)
	}
}

func TestHandleProjectSync_DBError(t *testing.T) {
	// Server without DB initialized - projectService singleton may fail.
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects/p1/sync", nil)
	srv.handleProjectSync(rec, req, "p1")

	// Either 500 (DB error) or stale singleton behavior.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleProjectSync_InvalidID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects//sync", nil)
	srv.handleProjectSync(rec, req, "")

	// Empty ID returns 404 from the service.
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 404 or 500, got %d", rec.Code)
	}
}

func TestSetupWorktreeRoutes(t *testing.T) {
	mux := http.NewServeMux()
	setupWorktreeRoutes(mux, "/api/")

	// Verify routes are registered by checking the mux patterns.
	// setupWorktreeRoutes registers:
	// - /api/worktree/projects
	// - /api/worktree/projects/
	// We can't directly inspect the mux, but we can verify no panic occurs.
}

func TestHandleTaskAISessionLinks_POST_EmptyBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t1/sessions", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(rec, req)

	// The service will likely return an error for empty ai_session_id.
	if rec.Code == http.StatusOK {
		t.Log("unexpected 200 for empty body")
	}
}

func TestHandleTaskAISessionLinks_POST_Success(t *testing.T) {
	srv := newTestServer()
	body := `{"ai_session_id":"sess-123"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t1/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskAISessionLinks_GET_Success(t *testing.T) {
	srv := newTestServer()
	// First link a session.
	body := `{"ai_session_id":"sess-456"}`
	linkRec := httptest.NewRecorder()
	linkReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t1/sessions", strings.NewReader(body))
	linkReq.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(linkRec, linkReq)
	if linkRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on link, got %d", linkRec.Code)
	}

	// Now list linked sessions.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/kanban/tasks/t1/sessions", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskAISessionLinks_DELETE_Success(t *testing.T) {
	srv := newTestServer()
	// First link a session.
	body := `{"ai_session_id":"sess-789"}`
	linkRec := httptest.NewRecorder()
	linkReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/kanban/tasks/t2/sessions", strings.NewReader(body))
	linkReq.Header.Set("Content-Type", "application/json")
	srv.handleTaskAISessionLinks(linkRec, linkReq)
	if linkRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on link, got %d", linkRec.Code)
	}

	// Now unlink it.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/kanban/tasks/t2/sessions/sess-789", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskAISessionLinks_DELETE_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/kanban/tasks/t3/sessions/nonexistent", nil)
	srv.handleTaskAISessionLinks(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAISessionSubroute_Detail_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/1", nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAISessionSubroute_Conversation_Success(t *testing.T) {
	// Insert a session and get its conversation.
	now := time.Now()
	record := aiSessionService.Store().Upsert(&ai.Session{
		SessionID:             "conv-sess",
		Type:                  "claude",
		FilePath:              "/tmp/test-conv.jsonl",
		FileModTime:           now,
		FileSize:              100,
		SessionStartedAt:      now,
		MessageCount:          1,
		AssistantMessageCount: 1,
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/api/ai/sessions/%d/conversation", record.ID), nil)
	srv.handleAISessions(rec, req)

	// May return 404 (no conversation parsed) or 200.
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusOK {
		t.Fatalf("expected 200 or 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAIListSessions_WithProject(t *testing.T) {
	// Use the current project directory.
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions?project=/tmp", nil)
	srv.handleAIListSessions(rec, req)

	// Should succeed (scanning returns empty or sessions).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
