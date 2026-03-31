package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleVersion(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	srv.handleVersion(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}

	var data map[string]string
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("parse data: %v", err)
	}
	if data["version"] == "" {
		t.Error("expected non-empty version")
	}
	if data["goVersion"] == "" {
		t.Error("expected non-empty goVersion")
	}
}
