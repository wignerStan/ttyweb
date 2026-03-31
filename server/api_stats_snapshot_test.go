package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/db"
	"ttyweb/service"
)

func TestGolden_Stats_ServiceUnavailable(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats", nil)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Stats_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/stats", nil)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Stats_Success(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats", nil)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Stats_CustomDays(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats?days=30", nil)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

func TestGolden_Stats_InvalidDays(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	srv.statsService = service.NewStatsService(gormDB)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats?days=abc", nil)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
