package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogWatcher_NewAndStop(t *testing.T) {
	watcher := NewLogWatcher("", 100*time.Millisecond)

	// Stopping before watching should not panic.
	watcher.Stop()
}

func TestLogWatcher_DoubleStop(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	watcher := NewLogWatcher("", 100*time.Millisecond)
	events := watcher.Watch()

	// Let the watcher do its initial scan.
	time.Sleep(200 * time.Millisecond)

	// First Stop should succeed.
	watcher.Stop()

	// Channel should be closed after first Stop.
	_, ok := <-events
	if ok {
		t.Error("event channel should be closed after first Stop()")
	}

	// Second Stop should not panic.
	watcher.Stop()
}

func TestLogWatcher_WatchLifecycle(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	watcher := NewLogWatcher("", 100*time.Millisecond)
	events := watcher.Watch()

	// Let the watcher do its initial scan (should find nothing).
	time.Sleep(200 * time.Millisecond)

	// Stop the watcher.
	watcher.Stop()

	// Channel should be closed.
	_, ok := <-events
	if ok {
		t.Error("event channel should be closed after Stop()")
	}
}

func TestLogWatcher_DetectsNewClaudeSession(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, ".claude", "projects", "-home-user-test")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)

	watcher := NewLogWatcher("/home/user/test", 100*time.Millisecond)
	events := watcher.Watch()

	// Wait for initial scan (empty).
	time.Sleep(200 * time.Millisecond)

	// Create a new session file.
	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"new-session"}`
	if err := os.WriteFile(filepath.Join(projectDir, "new-session.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for the watcher to detect the new file.
	select {
	case event := <-events:
		if event.Type != SessionEventNew {
			t.Errorf("expected event type %q, got %q", SessionEventNew, event.Type)
		}
		if event.Session.SessionID != "new-session" {
			t.Errorf("expected session ID %q, got %q", "new-session", event.Session.SessionID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for new session event")
	}

	watcher.Stop()
}

func TestLogWatcher_DetectsUpdatedSession(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, ".claude", "projects", "-home-user-test")
	filePath := filepath.Join(projectDir, "update-session.jsonl")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create initial session file.
	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"update-session"}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)

	watcher := NewLogWatcher("/home/user/test", 100*time.Millisecond)
	events := watcher.Watch()

	// Drain the initial "new" event.
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for initial event")
	}

	// Modify the file.
	time.Sleep(100 * time.Millisecond) // Ensure different mod time.
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"type":"user","message":{"role":"user","content":"Follow up"},"timestamp":"2025-12-01T10:01:00Z","sessionId":"update-session"}` + "\n"); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	_ = f.Close()

	// Wait for the update event.
	select {
	case event := <-events:
		if event.Type != SessionEventUpdated {
			t.Errorf("expected event type %q, got %q", SessionEventUpdated, event.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for update event")
	}

	watcher.Stop()
}

func TestLogWatcher_DetectsCompletedSession(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, ".claude", "projects", "-home-user-test")
	filePath := filepath.Join(projectDir, "completed-session.jsonl")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"completed-session"}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)

	watcher := NewLogWatcher("/home/user/test", 100*time.Millisecond)
	events := watcher.Watch()

	// Drain the initial "new" event.
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for initial event")
	}

	// Remove the file.
	time.Sleep(100 * time.Millisecond)
	if err := os.Remove(filePath); err != nil {
		t.Fatal(err)
	}

	// Wait for the completed event.
	select {
	case event := <-events:
		if event.Type != SessionEventCompleted {
			t.Errorf("expected event type %q, got %q", SessionEventCompleted, event.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for completed event")
	}

	watcher.Stop()
}

func TestLogWatcher_GetSessions(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, ".claude", "projects", "-home-user-test")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `{"type":"user","message":{"role":"user","content":"Hello!"},"timestamp":"2025-12-01T10:00:00Z","sessionId":"sess-a"}`
	if err := os.WriteFile(filepath.Join(projectDir, "sess-a.jsonl"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", homeDir)

	watcher := NewLogWatcher("/home/user/test", 100*time.Millisecond)
	events := watcher.Watch()

	// Wait for initial scan.
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for initial event")
	}

	sessions := watcher.GetSessions()
	if len(sessions) != 1 {
		t.Errorf("GetSessions() returned %d sessions, want 1", len(sessions))
	}
	if len(sessions) > 0 && sessions[0].SessionID != "sess-a" {
		t.Errorf("session ID = %q, want %q", sessions[0].SessionID, "sess-a")
	}

	watcher.Stop()
}
