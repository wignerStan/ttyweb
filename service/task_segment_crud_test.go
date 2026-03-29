package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ttyweb/db"
)

func setupSegmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := d.AutoMigrate(
		&db.TaskSegment{},
		&db.ChatMessage{},
		&db.CommandRecord{},
		&db.TaskSummary{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return d
}

func TestYearMonth(t *testing.T) {
	t.Parallel()

	tm := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	y, m := yearMonth(tm)
	if y != 2026 || m != 1 {
		t.Errorf("expected (2026, 1), got (%d, %d)", y, m)
	}

	tm = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	y, m = yearMonth(tm)
	if y != 2025 || m != 12 {
		t.Errorf("expected (2025, 12), got (%d, %d)", y, m)
	}
}

func TestCreateSegment(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, err := svc.CreateSegment("mysession", "mywindow", 0, "Test Task")
	if err != nil {
		t.Fatalf("CreateSegment: %v", err)
	}
	if segment.ID == "" {
		t.Error("expected non-empty ID")
	}
	if segment.SessionName != "mysession" {
		t.Errorf("expected session 'mysession', got %q", segment.SessionName)
	}
	if segment.WindowName != "mywindow" {
		t.Errorf("expected window 'mywindow', got %q", segment.WindowName)
	}
	if segment.PaneIndex != 0 {
		t.Errorf("expected pane index 0, got %d", segment.PaneIndex)
	}
	if segment.TaskTitle != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", segment.TaskTitle)
	}
	if segment.TaskStatus != StatusInProgress {
		t.Errorf("expected status %q, got %q", StatusInProgress, segment.TaskStatus)
	}
	if segment.StartedAt == nil {
		t.Error("expected non-nil StartedAt")
	}
}

func TestCreateSegment_EmptyTitle(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	_, err := svc.CreateSegment("s", "w", 0, "")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if err.Error() != "task_title is required" {
		t.Errorf("expected 'task_title is required', got %q", err.Error())
	}
}

func TestListSegments_All(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	_, _ = svc.CreateSegment("s1", "w", 0, "Task A")
	_, _ = svc.CreateSegment("s2", "w", 0, "Task B")

	segments, err := svc.ListSegments("")
	if err != nil {
		t.Fatalf("ListSegments: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	if segments[0].TaskTitle != "Task B" {
		t.Errorf("expected first 'Task B', got %q", segments[0].TaskTitle)
	}
}

func TestListSegments_BySession(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	_, _ = svc.CreateSegment("session-a", "w", 0, "Task A")
	_, _ = svc.CreateSegment("session-b", "w", 0, "Task B")
	_, _ = svc.CreateSegment("session-a", "w", 1, "Task C")

	segments, err := svc.ListSegments("session-a")
	if err != nil {
		t.Fatalf("ListSegments: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments for session-a, got %d", len(segments))
	}
}

func TestUpdateSegment(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	created, _ := svc.CreateSegment("s", "w", 0, "Original")

	newTitle := "Updated"
	updated, err := svc.UpdateSegment(created.ID, &newTitle, "")
	if err != nil {
		t.Fatalf("UpdateSegment: %v", err)
	}
	if updated.TaskTitle != "Updated" {
		t.Errorf("expected title 'Updated', got %q", updated.TaskTitle)
	}
	if updated.TaskStatus != StatusInProgress {
		t.Errorf("expected status unchanged, got %q", updated.TaskStatus)
	}

	updated, err = svc.UpdateSegment(created.ID, nil, StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateSegment: %v", err)
	}
	if updated.TaskStatus != StatusCompleted {
		t.Errorf("expected status %q, got %q", StatusCompleted, updated.TaskStatus)
	}
	if updated.CompletedAt == nil {
		t.Error("expected non-nil CompletedAt")
	}
}

func TestUpdateSegment_BothFields(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	created, _ := svc.CreateSegment("s", "w", 0, "Original")

	newTitle := "Both Updated"
	updated, err := svc.UpdateSegment(created.ID, &newTitle, StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateSegment: %v", err)
	}
	if updated.TaskTitle != "Both Updated" {
		t.Errorf("expected title 'Both Updated', got %q", updated.TaskTitle)
	}
	if updated.TaskStatus != StatusCompleted {
		t.Errorf("expected status %q, got %q", StatusCompleted, updated.TaskStatus)
	}
}

func TestUpdateSegment_InvalidStatus(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)
	created, _ := svc.CreateSegment("s", "w", 0, "Task")

	_, err := svc.UpdateSegment(created.ID, nil, "invalid_status")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestUpdateSegment_NoFields(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)
	created, _ := svc.CreateSegment("s", "w", 0, "Task")

	_, err := svc.UpdateSegment(created.ID, nil, "")
	if err == nil {
		t.Fatal("expected error when no fields provided")
	}
}

func TestUpdateSegment_NotFound(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	newTitle := "Ghost"
	_, err := svc.UpdateSegment("nonexistent-id", &newTitle, "")
	if err == nil {
		t.Fatal("expected error for nonexistent segment")
	}
}

func TestGetSegmentDetail(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Detail Task")
	_, _ = svc.AddMessage(segment.ID, RoleUser, "User message")
	_, _ = svc.AddCommandRecord(segment.ID, "echo hi", 0)

	detail, err := svc.GetSegmentDetail(segment.ID)
	if err != nil {
		t.Fatalf("GetSegmentDetail: %v", err)
	}
	if detail.Segment == nil {
		t.Fatal("expected non-nil segment")
	}
	if detail.Segment.TaskTitle != "Detail Task" {
		t.Errorf("expected title 'Detail Task', got %q", detail.Segment.TaskTitle)
	}
	if len(detail.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(detail.Messages))
	}
	if len(detail.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(detail.Commands))
	}
	if detail.Summary != nil {
		t.Error("expected nil summary when none exists")
	}
}

func TestGetSegmentDetail_NotFound(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	_, err := svc.GetSegmentDetail("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent segment")
	}
}

func TestGetSegmentDetail_WithSummary(t *testing.T) {
	t.Parallel()

	database := setupSegmentTestDB(t)
	svc := NewTaskSegmentService(database)

	segment, _ := svc.CreateSegment("s", "w", 0, "Summary Task")

	now := time.Now()
	summary := &db.TaskSummary{
		ID: "summary-1", Year: 2026, Mon: 3,
		SegmentID: segment.ID, SessionName: "s",
		WindowIndex: 0, WindowName: "w",
		CommandSummary: "Ran commands", OutputSummary: "Passed",
		SummaryStatus: "completed", GeneratedAt: &now,
	}
	if err := database.Create(summary).Error; err != nil {
		t.Fatalf("create summary: %v", err)
	}

	detail, err := svc.GetSegmentDetail(segment.ID)
	if err != nil {
		t.Fatalf("GetSegmentDetail: %v", err)
	}
	if detail.Summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if detail.Summary.CommandSummary != "Ran commands" {
		t.Errorf("expected 'Ran commands', got %q", detail.Summary.CommandSummary)
	}
}
