package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTaskStats_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	srv := newTestServerWithDB(t)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/stats", nil)
	rec := httptest.NewRecorder()
	srv.handleTaskStats(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTaskStats_ServiceUnavailable(t *testing.T) {
	t.Parallel()
	srv := newTestServer()
	// statsService is nil by default.

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats", nil)
	rec := httptest.NewRecorder()
	srv.handleTaskStats(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["success"] != false {
		t.Errorf("expected success=false, got %v", body["success"])
	}
}
