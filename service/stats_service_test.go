package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ttyweb/db"
)

func setupStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := d.AutoMigrate(&db.TaskSegment{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return d
}

func seedSegments(t *testing.T, gormDB *gorm.DB, segments []db.TaskSegment) {
	t.Helper()
	for i := range segments {
		if err := gormDB.Create(&segments[i]).Error; err != nil {
			t.Fatalf("seed segment %d: %v", i, err)
		}
	}
}

func TestStatsService_GetTaskStats(t *testing.T) {
	t.Parallel()

	database := setupStatsTestDB(t)
	svc := NewStatsService(database)

	now := time.Now()
	segments := []db.TaskSegment{
		{ID: "seg1", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T1", TaskStatus: "completed", CompletedAt: &now},
		{ID: "seg2", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T2", TaskStatus: "completed", CompletedAt: &now},
		{ID: "seg3", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T3", TaskStatus: "completed", CompletedAt: &now},
		{ID: "seg4", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T4", TaskStatus: "in_progress"},
		{ID: "seg5", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T5", TaskStatus: "failed"},
	}
	seedSegments(t, database, segments)

	stats, err := svc.GetTaskStats()
	if err != nil {
		t.Fatalf("GetTaskStats: %v", err)
	}
	if stats.Total != 5 {
		t.Errorf("expected Total=5, got %d", stats.Total)
	}
	if stats.Completed != 3 {
		t.Errorf("expected Completed=3, got %d", stats.Completed)
	}
	if stats.InProgress != 1 {
		t.Errorf("expected InProgress=1, got %d", stats.InProgress)
	}
	if stats.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", stats.Failed)
	}
}

func TestStatsService_GetTaskStats_Empty(t *testing.T) {
	t.Parallel()

	database := setupStatsTestDB(t)
	svc := NewStatsService(database)

	stats, err := svc.GetTaskStats()
	if err != nil {
		t.Fatalf("GetTaskStats: %v", err)
	}
	if stats.Total != 0 {
		t.Errorf("expected Total=0, got %d", stats.Total)
	}
	if stats.Completed != 0 {
		t.Errorf("expected Completed=0, got %d", stats.Completed)
	}
	if stats.InProgress != 0 {
		t.Errorf("expected InProgress=0, got %d", stats.InProgress)
	}
	if stats.Failed != 0 {
		t.Errorf("expected Failed=0, got %d", stats.Failed)
	}
}

func TestStatsService_GetDailyCompletions(t *testing.T) {
	t.Parallel()

	database := setupStatsTestDB(t)
	svc := NewStatsService(database)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	segments := []db.TaskSegment{
		{ID: "c1", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "C1", TaskStatus: "completed", CompletedAt: &today},
		{ID: "c2", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "C2", TaskStatus: "completed", CompletedAt: &today},
		{ID: "c3", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "C3", TaskStatus: "completed", CompletedAt: &today},
		{ID: "c4", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "C4", TaskStatus: "completed", CompletedAt: &yesterday},
		{ID: "c5", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "C5", TaskStatus: "completed", CompletedAt: &yesterday},
		{ID: "c6", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "C6", TaskStatus: "completed", CompletedAt: &twoDaysAgo},
		// In-progress task should NOT count.
		{ID: "ip1", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "IP1", TaskStatus: "in_progress"},
	}
	seedSegments(t, database, segments)

	daily, err := svc.GetDailyCompletions(7)
	if err != nil {
		t.Fatalf("GetDailyCompletions: %v", err)
	}
	if len(daily) != 3 {
		t.Fatalf("expected 3 days, got %d", len(daily))
	}

	// Results should be in descending order.
	if daily[0].Count != 3 {
		t.Errorf("expected today count=3, got %d", daily[0].Count)
	}
	if daily[1].Count != 2 {
		t.Errorf("expected yesterday count=2, got %d", daily[1].Count)
	}
	if daily[2].Count != 1 {
		t.Errorf("expected twoDaysAgo count=1, got %d", daily[2].Count)
	}
}

func TestStatsService_GetDailyCompletions_Limit(t *testing.T) {
	t.Parallel()

	database := setupStatsTestDB(t)
	svc := NewStatsService(database)

	now := time.Now()
	// Seed 15 days of data.
	for i := 0; i < 15; i++ {
		day := time.Date(now.Year(), now.Month(), now.Day()-i, 12, 0, 0, 0, now.Location())
		segments := []db.TaskSegment{
			{ID: "c" + string(rune('A'+i)), Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "Task", TaskStatus: "completed", CompletedAt: &day},
		}
		seedSegments(t, database, segments)
	}

	daily, err := svc.GetDailyCompletions(7)
	if err != nil {
		t.Fatalf("GetDailyCompletions: %v", err)
	}
	if len(daily) != 7 {
		t.Fatalf("expected 7 days, got %d", len(daily))
	}
}

func TestStatsService_GetDailyCompletions_Empty(t *testing.T) {
	t.Parallel()

	database := setupStatsTestDB(t)
	svc := NewStatsService(database)

	daily, err := svc.GetDailyCompletions(7)
	if err != nil {
		t.Fatalf("GetDailyCompletions: %v", err)
	}
	if len(daily) != 0 {
		t.Errorf("expected 0 days, got %d", len(daily))
	}
}

func TestStatsService_GetStatusBreakdown(t *testing.T) {
	t.Parallel()

	database := setupStatsTestDB(t)
	svc := NewStatsService(database)

	now := time.Now()
	segments := []db.TaskSegment{
		{ID: "s1", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T1", TaskStatus: "completed", CompletedAt: &now},
		{ID: "s2", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T2", TaskStatus: "completed", CompletedAt: &now},
		{ID: "s3", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T3", TaskStatus: "in_progress"},
		{ID: "s4", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T4", TaskStatus: "failed"},
		{ID: "s5", Year: 2026, Mon: 3, SessionName: "s", TaskTitle: "T5", TaskStatus: "cancelled"},
	}
	seedSegments(t, database, segments)

	breakdown, err := svc.GetStatusBreakdown()
	if err != nil {
		t.Fatalf("GetStatusBreakdown: %v", err)
	}
	if breakdown["completed"] != 2 {
		t.Errorf("expected completed=2, got %d", breakdown["completed"])
	}
	if breakdown["in_progress"] != 1 {
		t.Errorf("expected in_progress=1, got %d", breakdown["in_progress"])
	}
	if breakdown["failed"] != 1 {
		t.Errorf("expected failed=1, got %d", breakdown["failed"])
	}
	if breakdown["cancelled"] != 1 {
		t.Errorf("expected cancelled=1, got %d", breakdown["cancelled"])
	}
}
