package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ttyweb/ai"
)

func TestAISessionService_Store(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	if svc.Store() == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestAISessionService_GetSessions(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	store := svc.Store()
	now := time.Now()

	store.Upsert(&ai.Session{
		SessionID: "s1", Type: "claude_code",
		ProjectPath: "/proj/a", FilePath: "/path/1.jsonl", FileModTime: now,
	})
	store.Upsert(&ai.Session{
		SessionID: "s2", Type: "claude_code",
		ProjectPath: "/proj/b", FilePath: "/path/2.jsonl", FileModTime: now,
	})

	all := svc.GetSessions("")
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}
	filtered := svc.GetSessions("/proj/a")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 session for /proj/a, got %d", len(filtered))
	}
}

func TestAISessionService_GetSession(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	record := svc.Store().Upsert(&ai.Session{
		SessionID: "gs", Type: "codex",
		FilePath: "/path/gs.jsonl", FileModTime: time.Now(),
	})

	found, ok := svc.GetSession(record.ID)
	if !ok {
		t.Fatal("expected to find session")
	}
	if found.SessionID != "gs" {
		t.Errorf("expected SessionID 'gs', got %q", found.SessionID)
	}

	_, ok = svc.GetSession(99999)
	if ok {
		t.Error("expected not to find nonexistent session")
	}
}

func TestAISessionService_GetConversation_NotFound(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	msgs, err := svc.GetConversation(99999)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msgs != nil {
		t.Error("expected nil messages for nonexistent session")
	}
}

func TestAISessionService_GetConversation_UnknownType(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	svc.Store().Upsert(&ai.Session{
		SessionID: "unknown-type", Type: "unknown_assistant",
		FilePath: "/path/unknown.jsonl", FileModTime: time.Now(),
	})

	msgs, err := svc.GetConversation(1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msgs != nil {
		t.Error("expected nil messages for unknown type")
	}
}

func TestAISessionService_GetConversation_ClaudeCode(t *testing.T) {
	t.Parallel()

	content := `{"type":"user","message":{"role":"user","content":"Hello"},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"Hi there!"},"timestamp":"2025-12-01T10:30:01.000Z","sessionId":"s1"}
`
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "claude-session.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write jsonl: %v", err)
	}

	svc := NewAISessionService()
	record := svc.Store().Upsert(&ai.Session{
		SessionID:   "conv-test",
		Type:        string(ai.AssistantTypeClaudeCode),
		FilePath:    jsonlPath,
		FileModTime: time.Now(),
	})

	msgs, err := svc.GetConversation(record.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected first role 'user', got %q", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected second role 'assistant', got %q", msgs[1].Role)
	}
}

func TestAISessionService_GetConversation_Codex(t *testing.T) {
	t.Parallel()

	content := `{"timestamp":"2025-12-01T10:30:00.000Z","type":"event_msg","payload":{"type":"user_message","message":"Hello Codex"}}
{"timestamp":"2025-12-01T10:30:01.000Z","type":"event_msg","payload":{"type":"agent_message","message":"Hello from Codex!"}}
`
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "codex-session.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write jsonl: %v", err)
	}

	svc := NewAISessionService()
	record := svc.Store().Upsert(&ai.Session{
		SessionID:   "codex-conv-test",
		Type:        string(ai.AssistantTypeCodex),
		FilePath:    jsonlPath,
		FileModTime: time.Now(),
	})

	msgs, err := svc.GetConversation(record.ID)
	if err != nil {
		t.Fatalf("GetConversation Codex: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected first role 'user', got %q", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected second role 'assistant', got %q", msgs[1].Role)
	}
}

func TestAISessionService_GetConversation_NonexistentFile(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	svc.Store().Upsert(&ai.Session{
		SessionID:   "no-file",
		Type:        string(ai.AssistantTypeClaudeCode),
		FilePath:    "/nonexistent/path/session.jsonl",
		FileModTime: time.Now(),
	})

	_, err := svc.GetConversation(1)
	if err == nil {
		t.Error("expected error for nonexistent conversation file")
	}
}

func TestAISessionService_RefreshSession_NotFound(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()
	msgs, err := svc.RefreshSession(99999)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msgs != nil {
		t.Error("expected nil messages for nonexistent session")
	}
}

func TestAISessionService_RefreshSession_WithFile(t *testing.T) {
	content := `{"type":"user","message":{"role":"user","content":"Refresh test"},"timestamp":"2025-12-01T10:30:00.000Z","sessionId":"s1"}
{"type":"assistant","message":{"role":"assistant","content":"Refreshed!"},"timestamp":"2025-12-01T10:30:01.000Z","sessionId":"s1"}
`
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "refresh-session.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write jsonl: %v", err)
	}

	svc := NewAISessionService()
	record := svc.Store().Upsert(&ai.Session{
		SessionID:   "refresh-test",
		Type:        string(ai.AssistantTypeClaudeCode),
		FilePath:    jsonlPath,
		FileModTime: time.Now(),
	})

	// RefreshSession calls scanSessionByFilePath which scans real dirs.
	// The test file won't be found, so sessions will be nil. It falls through
	// to GetConversation which parses the file.
	msgs, err := svc.RefreshSession(record.ID)
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestAISessionService_CleanupStaleSessions(t *testing.T) {
	t.Parallel()

	svc := NewAISessionService()

	// CleanupStaleSessions calls scanAllSessions which scans real directories.
	// It should not panic and returns a count.
	_ = svc.CleanupStaleSessions()
}

func TestScanSessionByFilePath_NotFound(t *testing.T) {
	sessions, err := scanSessionByFilePath("/nonexistent/path/session.jsonl")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if sessions != nil {
		t.Errorf("expected nil sessions, got %v", sessions)
	}
}

// TestScanAllSessions verifies that scanAllSessions runs without panicking.
func TestScanAllSessions(t *testing.T) {
	sessions := scanAllSessions()
	// In a test environment this may or may not find sessions depending
	// on the test runner's home directory state. Just verify it returns.
	_ = sessions
}
