package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckEndpoint(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealthCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["success"] != true {
		t.Errorf("expected success=true, got %v", body["success"])
	}
}
