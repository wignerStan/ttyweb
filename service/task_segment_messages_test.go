package service

import (
	"testing"
)

func TestAddMessage(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")

	msg, err := svc.AddMessage(segment.ID, RoleUser, "Hello, world")
	if err != nil {
		t.Fatalf("AddMessage: %v", err)
	}
	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}
	if msg.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, msg.Role)
	}
	if msg.Content != "Hello, world" {
		t.Errorf("expected content 'Hello, world', got %q", msg.Content)
	}
	if msg.MsgTime == nil {
		t.Error("expected non-nil MsgTime")
	}
}

func TestAddMessage_InvalidRole(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")

	_, err := svc.AddMessage(segment.ID, "system", "bad role")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestAddMessage_AssistantRole(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")

	msg, err := svc.AddMessage(segment.ID, RoleAssistant, "I can help!")
	if err != nil {
		t.Fatalf("AddMessage assistant: %v", err)
	}
	if msg.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, msg.Role)
	}
}

func TestListMessages(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")
	_, _ = svc.AddMessage(segment.ID, RoleUser, "First")
	_, _ = svc.AddMessage(segment.ID, RoleAssistant, "Second")

	messages, err := svc.ListMessages(segment.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Content != "First" {
		t.Errorf("expected first 'First', got %q", messages[0].Content)
	}
	if messages[1].Content != "Second" {
		t.Errorf("expected second 'Second', got %q", messages[1].Content)
	}
}

func TestListMessages_Empty(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	messages, err := svc.ListMessages("nonexistent-id")
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestAddCommandRecord(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")

	record, err := svc.AddCommandRecord(segment.ID, "ls -la", 0)
	if err != nil {
		t.Fatalf("AddCommandRecord: %v", err)
	}
	if record.ID == "" {
		t.Error("expected non-empty record ID")
	}
	if record.Command != "ls -la" {
		t.Errorf("expected command 'ls -la', got %q", record.Command)
	}
	if record.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", record.ExitCode)
	}
	if record.CmdTime == nil {
		t.Error("expected non-nil CmdTime")
	}
}

func TestAddCommandRecord_WithNonZeroExitCode(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")

	record, err := svc.AddCommandRecord(segment.ID, "false", 1)
	if err != nil {
		t.Fatalf("AddCommandRecord: %v", err)
	}
	if record.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", record.ExitCode)
	}
}

func TestAddCommandRecord_EmptyCommand(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")

	_, err := svc.AddCommandRecord(segment.ID, "", 0)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestListCommands(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Task")
	_, _ = svc.AddCommandRecord(segment.ID, "git status", 0)
	_, _ = svc.AddCommandRecord(segment.ID, "git diff", 1)

	commands, err := svc.ListCommands(segment.ID)
	if err != nil {
		t.Fatalf("ListCommands: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if commands[0].Command != "git status" {
		t.Errorf("expected first 'git status', got %q", commands[0].Command)
	}
	if commands[1].ExitCode != 1 {
		t.Errorf("expected second exit code 1, got %d", commands[1].ExitCode)
	}
}

func TestListCommands_Empty(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	commands, err := svc.ListCommands("nonexistent-id")
	if err != nil {
		t.Fatalf("ListCommands: %v", err)
	}
	if len(commands) != 0 {
		t.Errorf("expected 0 commands, got %d", len(commands))
	}
}
