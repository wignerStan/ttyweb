package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ttyweb/ai"
	"ttyweb/db"
)

// setupSummaryTestDB creates an in-memory SQLite database with all models migrated.
func setupSummaryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := d.AutoMigrate(
		&db.TaskSegment{},
		&db.TaskSummary{},
		&db.ChatMessage{},
		&db.CommandRecord{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return d
}

// newMockAIServer creates an httptest.Server that responds to POST /v1/chat/completions
// with a canned OpenAI-format response. If statusCode is non-zero, it returns that status.
func newMockAIServer(t *testing.T, cannedResponse string, statusCode int) *httptest.Server {
	t.Helper()
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it is a POST to the completions endpoint.
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Verify system prompt is present in the request body.
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		hasSystem := false
		for _, m := range body.Messages {
			if m.Role == "system" {
				hasSystem = true
			}
		}
		if !hasSystem {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if statusCode != 0 {
			w.WriteHeader(statusCode)
			return
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": cannedResponse}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestSummaryService_GenerateSummary(t *testing.T) {
	t.Parallel()

	database := setupSummaryTestDB(t)
	mockSrv := newMockAIServer(t, "Built and tested auth module", 0)
	defer mockSrv.Close()

	aiClient := ai.NewClient("test-key", mockSrv.URL, "test-model")
	svc := NewSummaryService(database, aiClient)

	// Seed a segment.
	seg := &db.TaskSegment{
		ID: "seg-001", Year: 2026, Mon: 3,
		SessionName: "session1", WindowName: "win1",
		WindowIndex: 0, PaneIndex: 0,
		TaskTitle:  "Build auth module",
		TaskStatus: "completed",
		StartedAt:  ptrTime(time.Now()),
	}
	if err := database.Create(seg).Error; err != nil {
		t.Fatalf("create segment: %v", err)
	}

	// Seed commands.
	cmd := &db.CommandRecord{
		ID: "cmd-001", Year: 2026, Mon: 3,
		SegmentID: "seg-001", Command: "go test ./auth/...",
		ExitCode: 0, CmdTime: ptrTime(time.Now()),
	}
	if err := database.Create(cmd).Error; err != nil {
		t.Fatalf("create command: %v", err)
	}

	// Seed messages.
	msg := &db.ChatMessage{
		ID: "msg-001", Year: 2026, Mon: 3,
		SegmentID: "seg-001", Role: "user",
		Content: "Run the tests", MsgTime: ptrTime(time.Now()),
	}
	if err := database.Create(msg).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}

	summary, err := svc.GenerateSummary(context.Background(), "seg-001")
	if err != nil {
		t.Fatalf("GenerateSummary: %v", err)
	}

	if summary.CommandSummary != "Built and tested auth module" {
		t.Errorf("expected 'Built and tested auth module', got %q", summary.CommandSummary)
	}
	if summary.SummaryStatus != "completed" {
		t.Errorf("expected status 'completed', got %q", summary.SummaryStatus)
	}
	if summary.SegmentID != "seg-001" {
		t.Errorf("expected segment_id 'seg-001', got %q", summary.SegmentID)
	}
	if summary.GeneratedAt == nil {
		t.Error("expected non-nil GeneratedAt")
	}

	// Verify persisted to DB.
	var found db.TaskSummary
	if err := database.Where("segment_id = ?", "seg-001").First(&found).Error; err != nil {
		t.Fatalf("summary not persisted: %v", err)
	}
	if found.CommandSummary != "Built and tested auth module" {
		t.Errorf("persisted summary mismatch: %q", found.CommandSummary)
	}
}

func TestSummaryService_GenerateSummary_NoSegment(t *testing.T) {
	t.Parallel()

	database := setupSummaryTestDB(t)
	mockSrv := newMockAIServer(t, "irrelevant", 0)
	defer mockSrv.Close()

	aiClient := ai.NewClient("test-key", mockSrv.URL, "test-model")
	svc := NewSummaryService(database, aiClient)

	_, err := svc.GenerateSummary(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent segment")
	}
	if !strings.Contains(err.Error(), "segment not found") {
		t.Errorf("expected 'segment not found' error, got %q", err.Error())
	}
}

func TestSummaryService_GenerateSummary_NoData(t *testing.T) {
	t.Parallel()

	database := setupSummaryTestDB(t)
	mockSrv := newMockAIServer(t, "No activity recorded", 0)
	defer mockSrv.Close()

	aiClient := ai.NewClient("test-key", mockSrv.URL, "test-model")
	svc := NewSummaryService(database, aiClient)

	// Seed segment with no commands or messages.
	seg := &db.TaskSegment{
		ID: "seg-empty", Year: 2026, Mon: 3,
		SessionName: "session1", WindowName: "win1",
		WindowIndex: 0, PaneIndex: 0,
		TaskTitle:  "Empty task",
		TaskStatus: "completed",
		StartedAt:  ptrTime(time.Now()),
	}
	if err := database.Create(seg).Error; err != nil {
		t.Fatalf("create segment: %v", err)
	}

	summary, err := svc.GenerateSummary(context.Background(), "seg-empty")
	if err != nil {
		t.Fatalf("GenerateSummary with no data: %v", err)
	}

	if summary.CommandSummary != "No activity recorded" {
		t.Errorf("expected 'No activity recorded', got %q", summary.CommandSummary)
	}
}

func TestSummaryService_GenerateSummary_AIError(t *testing.T) {
	t.Parallel()

	database := setupSummaryTestDB(t)
	mockSrv := newMockAIServer(t, "", http.StatusInternalServerError)
	defer mockSrv.Close()

	aiClient := ai.NewClient("test-key", mockSrv.URL, "test-model")
	svc := NewSummaryService(database, aiClient)

	// Seed segment.
	seg := &db.TaskSegment{
		ID: "seg-err", Year: 2026, Mon: 3,
		SessionName: "session1", WindowName: "win1",
		WindowIndex: 0, PaneIndex: 0,
		TaskTitle:  "Error task",
		TaskStatus: "completed",
		StartedAt:  ptrTime(time.Now()),
	}
	if err := database.Create(seg).Error; err != nil {
		t.Fatalf("create segment: %v", err)
	}

	_, err := svc.GenerateSummary(context.Background(), "seg-err")
	if err == nil {
		t.Fatal("expected error when AI returns 500")
	}
	if !strings.Contains(err.Error(), "AI generation failed") {
		t.Errorf("expected 'AI generation failed' error, got %q", err.Error())
	}
}

func TestSummaryService_BuildPrompt(t *testing.T) {
	t.Parallel()

	commands := []db.CommandRecord{
		{Command: "go test ./...", ExitCode: 0},
		{Command: "git commit -m 'feat: add auth'", ExitCode: 0},
	}
	messages := []db.ChatMessage{
		{Role: "user", Content: "Add authentication"},
		{Role: "assistant", Content: "I'll implement JWT auth"},
	}

	prompt := buildPrompt("Implement authentication", commands, messages)

	if !strings.Contains(prompt, "Implement authentication") {
		t.Error("prompt should contain task title")
	}
	if !strings.Contains(prompt, "go test ./...") {
		t.Error("prompt should contain commands")
	}
	if !strings.Contains(prompt, "git commit") {
		t.Error("prompt should contain git command")
	}
	if !strings.Contains(prompt, "Add authentication") {
		t.Error("prompt should contain user message")
	}
	if !strings.Contains(prompt, "I'll implement JWT auth") {
		t.Error("prompt should contain assistant message")
	}
}

func TestSummaryService_BuildPrompt_WithExitCode(t *testing.T) {
	t.Parallel()

	commands := []db.CommandRecord{
		{Command: "go build ./...", ExitCode: 1},
	}
	prompt := buildPrompt("Build failed", commands, nil)

	if !strings.Contains(prompt, "(exit 1)") {
		t.Errorf("expected exit code in prompt, got:\n%s", prompt)
	}
}

func TestSummaryService_BuildPrompt_Empty(t *testing.T) {
	t.Parallel()

	prompt := buildPrompt("Empty task", nil, nil)

	if !strings.Contains(prompt, "Empty task") {
		t.Error("prompt should contain task title")
	}
	if strings.Contains(prompt, "Commands executed:") {
		t.Error("prompt should not contain commands section")
	}
	if strings.Contains(prompt, "AI Conversation:") {
		t.Error("prompt should not contain messages section")
	}
}

// ptrTime returns a pointer to the given time.
func ptrTime(t time.Time) *time.Time {
	return &t
}
