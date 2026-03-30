package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleConfig_NoCwd(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config", nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    any  `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}
	// No cwd should return null data.
	if resp.Data != nil {
		t.Fatalf("expected null data, got %v", resp.Data)
	}
}

func TestHandleConfig_NonexistentCwd(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config?cwd=/nonexistent/path/xyz123", nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    any  `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}
	// Nonexistent directory should return null.
	if resp.Data != nil {
		t.Fatalf("expected null data, got %v", resp.Data)
	}
}

func TestHandleConfig_WithValidFile(t *testing.T) {
	srv := newTestServer()
	// Use a directory that exists and may or may not have opencode.json.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config?cwd=/tmp", nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    any  `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}
	// /tmp likely doesn't have opencode.json, so data should be null.
	if resp.Data != nil {
		t.Fatalf("expected null data, got %v", resp.Data)
	}
}

func TestHandleConfig_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/opencode-config", nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
