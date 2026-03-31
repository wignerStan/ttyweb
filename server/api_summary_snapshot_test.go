package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ttyweb/db"
	"ttyweb/service"
)

func TestGolden_Summarize_ServiceUnavailable(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/seg-001/summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Summarize_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/seg-001/summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Summarize_MissingSegmentID(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.summaryService = service.NewSummaryService(gormDB, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments//summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Summarize_SegmentNotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.summaryService = service.NewSummaryService(gormDB, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/nonexistent/summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_GetSummary_ServiceUnavailable(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/seg-001/summary", nil)
	srv.handleGetSummary(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_GetSummary_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/seg-001/summary", nil)
	srv.handleGetSummary(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_GetSummary_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.summaryService = service.NewSummaryService(gormDB, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/nonexistent/summary", nil)
	srv.handleGetSummary(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ListSummaries_ServiceUnavailable(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ListSummaries_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ListSummaries_Empty(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.summaryService = service.NewSummaryService(gormDB, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_ListSummaries_WithData(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.summaryService = service.NewSummaryService(gormDB, nil)

	// Insert a summary directly into the DB with a fixed timestamp.
	fixedTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	summary := db.TaskSummary{
		ID:             "golden-summary-001",
		Year:           fixedTime.Year(),
		Mon:            int(fixedTime.Month()),
		SegmentID:      "seg-001",
		SessionName:    "test-session",
		WindowIndex:    0,
		WindowName:     "main",
		CommandSummary: "Implemented feature X",
		SummaryStatus:  "completed",
		GeneratedAt:    &fixedTime,
	}
	if err := gormDB.Create(&summary).Error; err != nil {
		t.Fatal(err)
	}
	// Override GORM's auto-set created_at for deterministic golden output.
	if err := gormDB.Model(&db.TaskSummary{}).Where("id = ?", summary.ID).
		Update("created_at", fixedTime).Error; err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
