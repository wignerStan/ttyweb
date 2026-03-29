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
	"os"
	"path/filepath"
	"testing"
	"time"

	"ttyweb/ai"
)

func TestHandleAIListSessions_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions", nil)
	srv.handleAIListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}

	// Data should be an array (possibly empty).
	var sessions []interface{}
	if err := json.Unmarshal(resp.Data, &sessions); err != nil {
		t.Fatalf("expected data to be an array, got: %s", string(resp.Data))
	}
}

func TestHandleAIListSessions_GET_WithProjectFilter(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions?project=/nonexistent", nil)
	srv.handleAIListSessions(rec, req)

	// Should succeed (scanning nonexistent project returns no sessions).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAISessionSubroute_Detail_WithStore(t *testing.T) {
	now := time.Now()

	// Insert a session into the global service store, then retrieve it.
	record := aiSessionService.Store().Upsert(ai.AISession{
		SessionID:             "test-sess",
		Type:                  "claude",
		FilePath:              "/tmp/test.jsonl",
		FileModTime:           now,
		FileSize:              100,
		SessionStartedAt:      now,
		MessageCount:          1,
		AssistantMessageCount: 0,
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/api/ai/sessions/%d", record.ID), nil)
	srv.handleAISessions(rec, req)

	_ = record

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
		t.Fatalf("expected success, got error")
	}
	if resp.Data["sessionId"] != "test-sess" {
		t.Fatalf("expected sessionId test-sess, got %v", resp.Data["sessionId"])
	}
}

func TestHandleAICleanupSessions_RemovesStale(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/cleanup", nil)
	srv.handleAISessions(rec, req)

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
		t.Fatalf("expected success, got error")
	}
	_, ok := resp.Data["removed"]
	if !ok {
		t.Fatal("expected 'removed' key in data")
	}
}

func TestHandleAISession_Conversation_Success(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Hello"},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"Hi there!"},"timestamp":"2025-12-01T10:30:01.000Z","sessionId":"s1"}
`
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "conv-session.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	record := aiSessionService.Store().Upsert(ai.AISession{
		SessionID:             "conv-success",
		Type:                  string(ai.AssistantTypeClaudeCode),
		FilePath:              jsonlPath,
		FileModTime:           now,
		FileSize:              int64(len(content)),
		SessionStartedAt:      now,
		MessageCount:          2,
		AssistantMessageCount: 1,
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/api/ai/sessions/%d/conversation", record.ID), nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}

	var msgs []map[string]interface{}
	if err := json.Unmarshal(resp.Data, &msgs); err != nil {
		t.Fatalf("expected array of messages, got: %s", string(resp.Data))
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestHandleAISession_Refresh_Success(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Refresh test"},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"Refreshed!"},"timestamp":"2025-12-01T10:30:01.000Z","sessionId":"s1"}
`
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "refresh-session.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	record := aiSessionService.Store().Upsert(ai.AISession{
		SessionID:             "refresh-success",
		Type:                  string(ai.AssistantTypeClaudeCode),
		FilePath:              jsonlPath,
		FileModTime:           now,
		FileSize:              int64(len(content)),
		SessionStartedAt:      now,
		MessageCount:          2,
		AssistantMessageCount: 1,
	})

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/api/ai/sessions/%d/refresh", record.ID), nil)
	srv.handleAISessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
