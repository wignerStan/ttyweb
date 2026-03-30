package service

import (
	"testing"
	"time"

	"ttyweb/ai"
)

func TestToRecord_FallbackFileModTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	session := ai.Session{
		SessionID: "sess-1", Type: string(ai.AssistantTypeClaudeCode),
		ProjectPath: "/home/user/project",
		FilePath:    "/home/user/.claude/projects/sess.jsonl",
		Model:       "claude-opus-4", Title: "Test Session",
		FileModTime: now, FileSize: 1024, MessageCount: 5,
	}

	record := toRecord(1, &session)
	if record.ID != 1 {
		t.Errorf("expected ID 1, got %d", record.ID)
	}
	if record.SessionID != "sess-1" {
		t.Errorf("expected SessionID 'sess-1', got %q", record.SessionID)
	}
	if record.SessionStartedAt != now {
		t.Error("expected SessionStartedAt to equal FileModTime when zero")
	}
	if record.FileSize != 1024 {
		t.Errorf("expected FileSize 1024, got %d", record.FileSize)
	}
}

func TestToRecord_WithSessionStartedAt(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	fileModTime := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	session := ai.Session{
		SessionID:        "sess-2",
		Type:             string(ai.AssistantTypeClaudeCode),
		SessionStartedAt: startedAt,
		FileModTime:      fileModTime,
	}

	record := toRecord(2, &session)
	if record.SessionStartedAt != startedAt {
		t.Error("expected SessionStartedAt to be set explicitly, not FileModTime")
	}
}

func TestToRecord_WithLastMessageAt(t *testing.T) {
	t.Parallel()

	lastMsg := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	session := ai.Session{
		SessionID:     "sess-3",
		Type:          "codex",
		LastMessageAt: &lastMsg,
	}

	record := toRecord(3, &session)
	if record.LastMessageAt == nil {
		t.Fatal("expected LastMessageAt to be set")
	}
	if !record.LastMessageAt.Equal(lastMsg) {
		t.Errorf("expected LastMessageAt %v, got %v", lastMsg, *record.LastMessageAt)
	}
}

func TestNewAISessionStore(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestAISessionStore_Upsert(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()
	session := ai.Session{
		SessionID:    "sess-upsert",
		Type:         "claude_code",
		ProjectPath:  "/home/user/proj",
		FilePath:     "/home/user/.claude/proj/sess.jsonl",
		Title:        "First upsert",
		MessageCount: 3,
		FileModTime:  time.Now(),
	}

	record := store.Upsert(&session)
	if record.ID != 1 {
		t.Errorf("expected first ID to be 1, got %d", record.ID)
	}
	if record.Title != "First upsert" {
		t.Errorf("expected title 'First upsert', got %q", record.Title)
	}

	// Upsert same session with updated data should preserve ID.
	session.Title = "Updated title"
	session.MessageCount = 7
	updated := store.Upsert(&session)
	if updated.ID != record.ID {
		t.Errorf("expected same ID on update, got %d vs %d", updated.ID, record.ID)
	}
	if updated.Title != "Updated title" {
		t.Errorf("expected title 'Updated title', got %q", updated.Title)
	}

	// Next insert should get a new ID.
	newSession := ai.Session{
		SessionID:   "sess-other",
		Type:        "claude_code",
		FilePath:    "/home/user/.claude/proj/other.jsonl",
		FileModTime: time.Now(),
	}
	newRecord := store.Upsert(&newSession)
	if newRecord.ID != 2 {
		t.Errorf("expected second ID to be 2, got %d", newRecord.ID)
	}
}

func TestAISessionStore_Upsert_DifferentType(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()

	claudeSession := ai.Session{
		SessionID: "same-id", Type: "claude_code",
		FilePath: "/path/claude.jsonl", FileModTime: time.Now(),
	}
	codexSession := ai.Session{
		SessionID: "same-id", Type: "codex",
		FilePath: "/path/codex.jsonl", FileModTime: time.Now(),
	}

	r1 := store.Upsert(&claudeSession)
	r2 := store.Upsert(&codexSession)

	if r1.ID == r2.ID {
		t.Errorf("expected different IDs for different types, got both %d", r1.ID)
	}
}

func TestAISessionStore_List(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()
	now := time.Now()
	msgTime1 := now.Add(-1 * time.Hour)
	msgTime2 := now.Add(-30 * time.Minute)
	msgTime3 := now

	store.Upsert(&ai.Session{
		SessionID: "sess-1", Type: "claude_code",
		ProjectPath: "/proj/a", FilePath: "/path/1.jsonl",
		LastMessageAt: &msgTime1, FileModTime: now,
	})
	store.Upsert(&ai.Session{
		SessionID: "sess-2", Type: "claude_code",
		ProjectPath: "/proj/b", FilePath: "/path/2.jsonl",
		LastMessageAt: &msgTime3, FileModTime: now,
	})
	store.Upsert(&ai.Session{
		SessionID: "sess-3", Type: "claude_code",
		ProjectPath: "/proj/a", FilePath: "/path/3.jsonl",
		LastMessageAt: &msgTime2, FileModTime: now,
	})

	// List all should be sorted by LastMessageAt descending.
	all := store.List("")
	if len(all) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(all))
	}
	if all[0].SessionID != "sess-2" {
		t.Errorf("expected first 'sess-2' (newest), got %q", all[0].SessionID)
	}
	if all[2].SessionID != "sess-1" {
		t.Errorf("expected last 'sess-1' (oldest), got %q", all[2].SessionID)
	}

	// Filter by project path.
	filtered := store.List("/proj/a")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 sessions for /proj/a, got %d", len(filtered))
	}
}

func TestAISessionStore_List_NilLastMessageAt(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()
	now := time.Now()
	msgTime := now.Add(-1 * time.Hour)

	store.Upsert(&ai.Session{
		SessionID: "no-msg-time", Type: "claude_code",
		FilePath:         "/path/a.jsonl",
		SessionStartedAt: now.Add(-2 * time.Hour), FileModTime: now.Add(-2 * time.Hour),
	})
	store.Upsert(&ai.Session{
		SessionID: "with-msg-time", Type: "claude_code",
		FilePath:      "/path/b.jsonl",
		LastMessageAt: &msgTime, FileModTime: now.Add(-1 * time.Hour),
	})

	all := store.List("")
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}
	// Session with LastMessageAt should come first; nil falls to SessionStartedAt.
	if all[0].SessionID != "with-msg-time" {
		t.Errorf("expected 'with-msg-time' first, got %q", all[0].SessionID)
	}

	// Both nil LastMessageAt: sort by SessionStartedAt descending.
	store.Upsert(&ai.Session{
		SessionID: "newer", Type: "claude_code",
		FilePath:         "/path/c.jsonl",
		SessionStartedAt: now.Add(-30 * time.Minute), FileModTime: now.Add(-30 * time.Minute),
	})
	all = store.List("")
	if all[0].SessionID != "with-msg-time" && all[0].SessionID != "newer" {
		t.Errorf("expected newest first, got %q", all[0].SessionID)
	}
}

func TestAISessionStore_GetByID(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()
	session := ai.Session{
		SessionID: "get-by-id", Type: "claude_code",
		FilePath: "/path/get.jsonl", Title: "Find Me", FileModTime: time.Now(),
	}
	inserted := store.Upsert(&session)

	found, ok := store.GetByID(inserted.ID)
	if !ok {
		t.Fatal("expected to find session by ID")
	}
	if found.Title != "Find Me" {
		t.Errorf("expected title 'Find Me', got %q", found.Title)
	}

	_, ok = store.GetByID(9999)
	if ok {
		t.Error("expected not to find session with ID 9999")
	}
}

func TestAISessionStore_DeleteMissingFiles(t *testing.T) {
	t.Parallel()

	store := NewAISessionStore()
	now := time.Now()

	store.Upsert(&ai.Session{
		SessionID: "keep", Type: "claude_code",
		FilePath: "/path/keep.jsonl", FileModTime: now,
	})
	store.Upsert(&ai.Session{
		SessionID: "remove", Type: "claude_code",
		FilePath: "/path/remove.jsonl", FileModTime: now,
	})
	store.Upsert(&ai.Session{
		SessionID: "also-remove", Type: "codex",
		FilePath: "/path/also-remove.jsonl", FileModTime: now,
	})

	// Keep only one.
	removed := store.DeleteMissingFiles(map[string]bool{"/path/keep.jsonl": true})
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}
	remaining := store.List("")
	if len(remaining) != 1 || remaining[0].SessionID != "keep" {
		t.Errorf("expected only 'keep', got %d sessions", len(remaining))
	}

	// Remove all remaining.
	removed = store.DeleteMissingFiles(map[string]bool{})
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if len(store.List("")) != 0 {
		t.Error("expected no remaining sessions")
	}
}
