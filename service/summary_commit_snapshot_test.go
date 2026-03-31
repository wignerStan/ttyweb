package service

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ttyweb/db"
)

var updateSummaryCommitGolden = flag.Bool("update-summary-commit", false, "update golden files for summary and commit message tests")

func compareSummaryCommitGolden(t *testing.T, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", t.Name()+".golden")
	if *updateSummaryCommitGolden {
		t.Logf("updating golden file: %s", golden)
		_ = os.MkdirAll(filepath.Dir(golden), 0o755)
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
	}
	want, err := os.ReadFile(golden) //nolint:gosec // test file from t.TempDir()
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("mismatch:\n got: %s\nwant: %s", got, want)
	}
}

// sanitizeSummaryJSON removes dynamic fields (generated_at, created_at) from JSON
// and replaces them with stable placeholders so golden files are deterministic.
func sanitizeSummaryJSON(data []byte) []byte {
	// Replace ISO 8601 timestamps in "generated_at" and "created_at" fields.
	re := regexp.MustCompile(`"generated_at":\s*"[^"]*"`)
	data = re.ReplaceAll(data, []byte(`"generated_at": "<time>"`))
	re = regexp.MustCompile(`"created_at":\s*"[^"]*"`)
	data = re.ReplaceAll(data, []byte(`"created_at": "<time>"`))
	return data
}

// setupSnapshotSummaryDB creates an in-memory SQLite DB with the TaskSummary model.
func setupSnapshotSummaryDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := d.AutoMigrate(&db.TaskSummary{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return d
}

// --- GetSummary snapshot tests ---

func TestSnapshot_GetSummary_NotFound(t *testing.T) {
	t.Parallel()

	database := setupSnapshotSummaryDB(t)
	svc := NewSummaryService(database, nil)

	result, err := svc.GetSummary("nonexistent-segment")
	if err != nil {
		t.Fatalf("expected no error for nonexistent segment, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got: %+v", result)
	}

	// Snapshot: empty JSON null representation.
	got := []byte("null\n")
	compareSummaryCommitGolden(t, got)
}

func TestSnapshot_GetSummary_Found(t *testing.T) {
	t.Parallel()

	database := setupSnapshotSummaryDB(t)
	svc := NewSummaryService(database, nil)

	now := time.Now().UTC().Truncate(time.Microsecond)
	summary := &db.TaskSummary{
		ID:             "sum-001",
		Year:           2026,
		Mon:            3,
		SegmentID:      "seg-001",
		SessionName:    "session1",
		WindowIndex:    0,
		WindowName:     "win1",
		CommandSummary: "Implemented auth module with JWT tokens",
		SummaryStatus:  "completed",
		GeneratedAt:    &now,
	}
	if err := database.Create(summary).Error; err != nil {
		t.Fatalf("create summary: %v", err)
	}

	result, err := svc.GetSummary("seg-001")
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	got, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	got = append(got, '\n')
	got = sanitizeSummaryJSON(got)
	compareSummaryCommitGolden(t, got)
}

// --- ListSummaries snapshot tests ---

func TestSnapshot_ListSummaries_Empty(t *testing.T) {
	t.Parallel()

	database := setupSnapshotSummaryDB(t)
	svc := NewSummaryService(database, nil)

	summaries, err := svc.ListSummaries()
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) != 0 {
		t.Fatalf("expected 0 summaries, got %d", len(summaries))
	}

	got, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')
	compareSummaryCommitGolden(t, got)
}

func TestSnapshot_ListSummaries_WithData(t *testing.T) {
	t.Parallel()

	database := setupSnapshotSummaryDB(t)
	svc := NewSummaryService(database, nil)

	now := time.Now().UTC().Truncate(time.Microsecond)
	earlier := now.Add(-1 * time.Hour)

	summary1 := &db.TaskSummary{
		ID:             "sum-001",
		Year:           2026,
		Mon:            3,
		SegmentID:      "seg-001",
		SessionName:    "session1",
		WindowIndex:    0,
		WindowName:     "win1",
		CommandSummary: "First task: added login page",
		SummaryStatus:  "completed",
		GeneratedAt:    &earlier,
	}
	summary2 := &db.TaskSummary{
		ID:             "sum-002",
		Year:           2026,
		Mon:            3,
		SegmentID:      "seg-002",
		SessionName:    "session1",
		WindowIndex:    1,
		WindowName:     "win2",
		CommandSummary: "Second task: added user registration",
		SummaryStatus:  "completed",
		GeneratedAt:    &now,
	}
	for _, s := range []*db.TaskSummary{summary1, summary2} {
		if err := database.Create(s).Error; err != nil {
			t.Fatalf("create summary: %v", err)
		}
	}

	summaries, err := svc.ListSummaries()
	if err != nil {
		t.Fatalf("ListSummaries: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	// Most recent first.
	if summaries[0].SegmentID != "seg-002" {
		t.Errorf("expected first summary to be seg-002 (most recent), got %q", summaries[0].SegmentID)
	}

	got, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')
	got = sanitizeSummaryJSON(got)
	compareSummaryCommitGolden(t, got)
}

// --- CommitMessageService snapshot tests ---

func TestSnapshot_NewCommitMessage_NoConfig(t *testing.T) {
	t.Parallel()

	svc := NewCommitMessageService(nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.client != nil {
		t.Fatal("expected nil client when config is nil")
	}

	// Snapshot the error message from GenerateCommitMessage with nil client.
	_, err := svc.GenerateCommitMessage(t.Context(), "some diff")
	if err == nil {
		t.Fatal("expected error for nil client")
	}

	got := []byte(err.Error() + "\n")
	compareSummaryCommitGolden(t, got)
}

func TestSnapshot_GenerateCommitMessage_EmptyDiff(t *testing.T) {
	t.Parallel()

	svc := NewCommitMessageService(nil)
	_, err := svc.GenerateCommitMessage(t.Context(), "")
	if err == nil {
		t.Fatal("expected error for empty diff")
	}

	got := []byte(err.Error() + "\n")
	compareSummaryCommitGolden(t, got)
}

func TestSnapshot_GenerateCommitMessage_NoClient(t *testing.T) {
	t.Parallel()

	svc := NewCommitMessageService(nil)
	_, err := svc.GenerateCommitMessage(t.Context(), "diff content here")
	if err == nil {
		t.Fatal("expected error when client is nil")
	}

	got := []byte(err.Error() + "\n")
	compareSummaryCommitGolden(t, got)
}
