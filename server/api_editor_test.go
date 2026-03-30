package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestHandleEditorOpen_Success(t *testing.T) {
	srv := newTestServer()
	body := `{"path":"/tmp/test.go","line":10,"col":5}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/editor/open", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleEditorOpen(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool              `json:"success"`
		Data    map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success")
	}
	if resp.Data["url"] == "" {
		t.Fatal("expected non-empty url")
	}
	if resp.Data["path"] == "" {
		t.Fatal("expected non-empty path")
	}
	// Verify URL starts with vscode://file.
	if resp.Data["url"][:13] != "vscode://file" {
		t.Fatalf("expected url to start with vscode://file, got %q", resp.Data["url"])
	}
	// Verify line and col are in the URL.
	if !contains(resp.Data["url"], ":10:5") {
		t.Fatalf("expected :10:5 in url, got %q", resp.Data["url"])
	}
}

func TestHandleEditorOpen_CursorPath(t *testing.T) {
	srv := newTestServer()
	body := `{"path":"/home/user/.cursor/project/main.go"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/editor/open", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleEditorOpen(rec, req)

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
	if resp.Data["url"][:13] != "cursor://file" {
		t.Fatalf("expected url to start with cursor://file, got %q", resp.Data["url"])
	}
	// Verify path contains .cursor.
	if !contains(resp.Data["url"], ".cursor") {
		t.Fatalf("expected .cursor in url, got %q", resp.Data["url"])
	}
}

func TestHandleEditorOpen_MissingPath(t *testing.T) {
	srv := newTestServer()
	body := `{"line":10}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/editor/open", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleEditorOpen(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleEditorOpen_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/editor/open", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleEditorOpen(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleEditorOpen_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/editor/open", nil)
	srv.handleEditorOpen(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleEditorOpen_NoLineCol(t *testing.T) {
	srv := newTestServer()
	body := `{"path":"/tmp/test.go"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/editor/open", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleEditorOpen(rec, req)

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
	// URL should not have line/col suffix.
	url := resp.Data["url"]
	if url != "" && url[len(url)-1] >= '0' && url[len(url)-1] <= '9' {
		t.Fatalf("expected no line/col in url, got %q", url)
	}
}
