package service

import (
	"testing"
)

func TestRequireFields_AllPresent(t *testing.T) {
	t.Parallel()

	err := requireFields("task_id", "t-1", "ai_session_id", "s-1")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestRequireFields_MissingValue(t *testing.T) {
	t.Parallel()

	err := requireFields("task_id", "", "ai_session_id", "s-1")
	if err == nil {
		t.Fatal("expected error for missing task_id, got nil")
	}
	if err.Error() != "task_id is required" {
		t.Errorf("expected 'task_id is required', got %q", err.Error())
	}
}

func TestRequireFields_SecondMissing(t *testing.T) {
	t.Parallel()

	err := requireFields("task_id", "t-1", "ai_session_id", "")
	if err == nil {
		t.Fatal("expected error for missing ai_session_id, got nil")
	}
	if err.Error() != "ai_session_id is required" {
		t.Errorf("expected 'ai_session_id is required', got %q", err.Error())
	}
}

func TestRequireFields_PanicsOnOddArgs(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for odd number of arguments")
		}
	}()

	// Calling with odd args triggers a panic (tested via recover above).
	//nolint:staticcheck
	_ = requireFields("only_label")
}

func TestNewTaskAISessionService(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestTaskAISessionService_LinkSession(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	link, err := svc.LinkSession("task-1", "session-1")
	if err != nil {
		t.Fatalf("LinkSession: %v", err)
	}
	if link.ID == "" {
		t.Error("expected non-empty link ID")
	}
	if link.TaskID != "task-1" {
		t.Errorf("expected TaskID 'task-1', got %q", link.TaskID)
	}
	if link.AISessionID != "session-1" {
		t.Errorf("expected AISessionID 'session-1', got %q", link.AISessionID)
	}
}

func TestTaskAISessionService_LinkSession_EmptyTaskID(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	_, err := svc.LinkSession("", "session-1")
	if err == nil {
		t.Fatal("expected error for empty task ID, got nil")
	}
}

func TestTaskAISessionService_LinkSession_EmptySessionID(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	_, err := svc.LinkSession("task-1", "")
	if err == nil {
		t.Fatal("expected error for empty session ID, got nil")
	}
}

func TestTaskAISessionService_LinkMultiple(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	_, err := svc.LinkSession("task-1", "session-1")
	if err != nil {
		t.Fatalf("LinkSession 1: %v", err)
	}
	_, err = svc.LinkSession("task-1", "session-2")
	if err != nil {
		t.Fatalf("LinkSession 2: %v", err)
	}
	_, err = svc.LinkSession("task-2", "session-1")
	if err != nil {
		t.Fatalf("LinkSession 3: %v", err)
	}

	links, err := svc.ListLinkedSessions("task-1")
	if err != nil {
		t.Fatalf("ListLinkedSessions: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links for task-1, got %d", len(links))
	}
}

func TestTaskAISessionService_UnlinkSession(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	_, _ = svc.LinkSession("task-1", "session-1")
	_, _ = svc.LinkSession("task-1", "session-2")

	err := svc.UnlinkSession("task-1", "session-1")
	if err != nil {
		t.Fatalf("UnlinkSession: %v", err)
	}

	links, _ := svc.ListLinkedSessions("task-1")
	if len(links) != 1 {
		t.Fatalf("expected 1 link after unlink, got %d", len(links))
	}
	if links[0].AISessionID != "session-2" {
		t.Errorf("expected remaining link to session-2, got %q", links[0].AISessionID)
	}
}

func TestTaskAISessionService_UnlinkSession_NotFound(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	err := svc.UnlinkSession("task-1", "session-nonexistent")
	if err == nil {
		t.Fatal("expected error for unlinking nonexistent link, got nil")
	}
	if err.Error() != "link not found" {
		t.Errorf("expected 'link not found', got %q", err.Error())
	}
}

func TestTaskAISessionService_UnlinkSession_EmptyTaskID(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	err := svc.UnlinkSession("", "session-1")
	if err == nil {
		t.Fatal("expected error for empty task ID, got nil")
	}
}

func TestTaskAISessionService_UnlinkSession_EmptySessionID(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	err := svc.UnlinkSession("task-1", "")
	if err == nil {
		t.Fatal("expected error for empty session ID, got nil")
	}
}

func TestTaskAISessionService_ListLinkedSessions(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	// Empty list for task with no links.
	links, err := svc.ListLinkedSessions("task-no-links")
	if err != nil {
		t.Fatalf("ListLinkedSessions: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("expected 0 links, got %d", len(links))
	}

	// Add some links.
	_, _ = svc.LinkSession("task-1", "s1")
	_, _ = svc.LinkSession("task-1", "s2")
	_, _ = svc.LinkSession("task-2", "s1")

	links, err = svc.ListLinkedSessions("task-1")
	if err != nil {
		t.Fatalf("ListLinkedSessions: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links for task-1, got %d", len(links))
	}

	links, err = svc.ListLinkedSessions("task-2")
	if err != nil {
		t.Fatalf("ListLinkedSessions: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link for task-2, got %d", len(links))
	}
}

func TestTaskAISessionService_ListLinkedSessions_EmptyTaskID(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	_, err := svc.ListLinkedSessions("")
	if err == nil {
		t.Fatal("expected error for empty task ID, got nil")
	}
}

func TestTaskAISessionService_LinkUnlinkReLink(t *testing.T) {
	t.Parallel()

	svc := NewTaskAISessionService()

	link1, _ := svc.LinkSession("task-1", "session-1")
	_ = svc.UnlinkSession("task-1", "session-1")

	link2, err := svc.LinkSession("task-1", "session-1")
	if err != nil {
		t.Fatalf("re-link after unlink: %v", err)
	}

	// IDs should be different (new link created).
	if link1.ID == link2.ID {
		t.Error("expected different IDs for re-created link")
	}

	links, _ := svc.ListLinkedSessions("task-1")
	if len(links) != 1 {
		t.Fatalf("expected 1 link after re-link, got %d", len(links))
	}
}
