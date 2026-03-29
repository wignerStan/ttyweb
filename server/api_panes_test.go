package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlePaneStatus_GET_Empty(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/panes/status", nil)
	srv.handlePaneStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(resp.Data))
	}
}

func TestHandlePaneStatus_PUT(t *testing.T) {
	srv := newTestServer()
	body := `{"pane_key":"%1","status":"running"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/panes/status", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handlePaneStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			PaneKey string `json:"pane_key"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.PaneKey != "%1" {
		t.Fatalf("expected pane_key %%1, got %q", resp.Data.PaneKey)
	}
	if resp.Data.Status != "running" {
		t.Fatalf("expected 'running', got %q", resp.Data.Status)
	}
}

func TestHandlePaneStatus_PUT_MissingPaneKey(t *testing.T) {
	srv := newTestServer()
	body := `{"status":"running"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/panes/status", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handlePaneStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePaneStatus_PUT_MissingStatus(t *testing.T) {
	srv := newTestServer()
	body := `{"pane_key":"%1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/panes/status", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handlePaneStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePaneStatus_PUT_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/panes/status", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handlePaneStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePaneStatus_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/panes/status", nil)
	srv.handlePaneStatus(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
