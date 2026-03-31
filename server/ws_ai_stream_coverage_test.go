package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/db"
	"ttyweb/service"
)

// TestBuiltinRoleIDMap verifies the role ID map has expected entries.
func TestBuiltinRoleIDMap(t *testing.T) {
	if builtinRoleIDMap["cli"] != 1 {
		t.Error("expected cli -> 1")
	}
	if builtinRoleIDMap["ops"] != 7 {
		t.Error("expected ops -> 7")
	}
	if builtinRoleIDMap["prompt"] != 5 {
		t.Error("expected prompt -> 5")
	}
}

func TestHandleTaskStats_Success(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats", nil)
	srv.handleTaskStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskStats_WithDaysParam(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats?days=30", nil)
	srv.handleTaskStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskStats_InvalidDaysParam(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats?days=notanumber", nil)
	srv.handleTaskStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskStats_NegativeDays(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats?days=-5", nil)
	srv.handleTaskStats(rec, req)

	// Negative days falls back to default (7), should succeed.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSummarizeSegment_MissingSegmentID_WithService(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewSummaryService(gormDB, nil)
	srv.summaryService = svc

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments//summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
