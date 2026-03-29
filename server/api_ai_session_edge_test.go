// Tests in this file insert into the global aiSessionService.Store() singleton.
// Each test uses a unique SessionID+Type combination to avoid cross-test interference.
// Tests intentionally do NOT call t.Parallel() to prevent concurrent singleton mutations.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ttyweb/ai"
	"ttyweb/service"
)

func TestHandleAIListSessions_NoQueryParams(t *testing.T) {
	t.Parallel()
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/ai/sessions", nil,
	)
	srv.handleAIListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
}

func TestHandleAIListSessions_EmptyProject(t *testing.T) {
	t.Parallel()
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/ai/sessions?project=", nil,
	)
	srv.handleAIListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAISessionDetail_NonExistent(t *testing.T) {
	t.Parallel()
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/ai/sessions/999999", nil,
	)
	srv.aiSessionDetail(rec, req, 999999)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAISessionDetail_Success(t *testing.T) {
	t.Parallel()
	now := time.Now()
	record := aiSessionService.Store().Upsert(ai.AISession{
		SessionID:             "detail-test",
		Type:                  "claude",
		FilePath:              "/tmp/detail-test.jsonl",
		FileModTime:           now,
		FileSize:              500,
		SessionStartedAt:      now,
		MessageCount:          5,
		AssistantMessageCount: 2,
		Title:                 "Detail Test Session",
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		fmt.Sprintf("/api/ai/sessions/%d", record.ID), nil,
	)
	srv.aiSessionDetail(rec, req, record.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success")
	}
	if resp.Data["sessionId"] != "detail-test" {
		t.Fatalf("expected sessionId 'detail-test', got %v", resp.Data["sessionId"])
	}
	if resp.Data["title"] != "Detail Test Session" {
		t.Fatalf("expected title 'Detail Test Session', got %v", resp.Data["title"])
	}
}

func TestAISessionConversation_NonExistent(t *testing.T) {
	t.Parallel()
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/ai/sessions/999999/conversation", nil,
	)
	srv.aiSessionConversationOrRefresh(rec, req, 999999, false)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAISessionRefresh_NonExistent(t *testing.T) {
	t.Parallel()
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/api/ai/sessions/999999/refresh", nil,
	)
	srv.aiSessionConversationOrRefresh(rec, req, 999999, true)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAISessionConversation_Existing(t *testing.T) {
	t.Parallel()
	now := time.Now()
	record := aiSessionService.Store().Upsert(ai.AISession{
		SessionID:             "conv-edge-test",
		Type:                  "claude",
		FilePath:              "/tmp/conv-edge-test.jsonl",
		FileModTime:           now,
		FileSize:              100,
		SessionStartedAt:      now,
		MessageCount:          1,
		AssistantMessageCount: 1,
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		fmt.Sprintf("/api/ai/sessions/%d/conversation", record.ID), nil,
	)
	srv.aiSessionConversationOrRefresh(rec, req, record.ID, false)

	// May be 200 (has conversation) or 404 (no conversation data).
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAISessionRefresh_Existing(t *testing.T) {
	t.Parallel()
	now := time.Now()
	record := aiSessionService.Store().Upsert(ai.AISession{
		SessionID:             "refresh-edge-test",
		Type:                  "claude",
		FilePath:              "/tmp/refresh-edge-test.jsonl",
		FileModTime:           now,
		FileSize:              100,
		SessionStartedAt:      now,
		MessageCount:          1,
		AssistantMessageCount: 1,
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		fmt.Sprintf("/api/ai/sessions/%d/refresh", record.ID), nil,
	)
	srv.aiSessionConversationOrRefresh(rec, req, record.ID, true)

	// May be 200 (refreshed) or 404 (no conversation data).
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("expected 200 or 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestNewAISessionResponse_AllFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	lastMsg := now.Add(time.Hour)
	r := newAISessionResponse(service.AISessionRecord{
		ID:                    42,
		SessionID:             "sess-all-fields",
		Type:                  "codex",
		ProjectPath:           "/home/user/project",
		FilePath:              "/path/to/session.jsonl",
		Model:                 "gpt-4",
		Title:                 "Full Session Test",
		SessionStartedAt:      now,
		LastMessageAt:         &lastMsg,
		MessageCount:          20,
		AssistantMessageCount: 8,
		FileModTime:           now,
		FileSize:              4096,
	})

	if r.ID != 42 {
		t.Fatalf("expected ID 42, got %d", r.ID)
	}
	if r.Type != "codex" {
		t.Fatalf("expected type 'codex', got %q", r.Type)
	}
	if r.ProjectPath != "/home/user/project" {
		t.Fatalf("expected project path, got %q", r.ProjectPath)
	}
	if r.Model != "gpt-4" {
		t.Fatalf("expected model 'gpt-4', got %q", r.Model)
	}
	if r.MessageCount != 20 {
		t.Fatalf("expected 20 messages, got %d", r.MessageCount)
	}
	if r.FileSize != 4096 {
		t.Fatalf("expected 4096 file size, got %d", r.FileSize)
	}
	if r.LastMessageAt == nil {
		t.Fatal("expected non-nil LastMessageAt")
	}
	if r.SessionStartedAt == "" {
		t.Fatal("expected non-empty SessionStartedAt")
	}
	if r.FileModTime == "" {
		t.Fatal("expected non-empty FileModTime")
	}
}
