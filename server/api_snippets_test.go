package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSnippets_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/snippets", nil)
	srv.handleSnippets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
}

func TestHandleSnippets_POST(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"List Files","command":"ls -la"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/snippets", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool    `json:"success"`
		Data    Snippet `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.Index != 0 {
		t.Fatalf("expected index 0, got %d", resp.Data.Index)
	}
	if resp.Data.Name != "List Files" {
		t.Fatalf("expected 'List Files', got %q", resp.Data.Name)
	}
}

func TestHandleSnippets_POST_MissingName(t *testing.T) {
	srv := newTestServer()
	body := `{"command":"ls"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/snippets", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippets(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSnippets_POST_MissingCommand(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"List Files"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/snippets", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippets(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSnippets_POST_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/snippets", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippets(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSnippets_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/snippets", nil)
	srv.handleSnippets(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSnippetDetail_PUT(t *testing.T) {
	srv := newTestServer()
	store.CreateSnippet(Snippet{Name: "Old", Command: "ls"})
	body := `{"name":"Updated","command":"cat"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/snippets/0", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool    `json:"success"`
		Data    Snippet `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.Name != "Updated" {
		t.Fatalf("expected 'Updated', got %q", resp.Data.Name)
	}
}

func TestHandleSnippetDetail_PUT_NotFound(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"X","command":"y"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/snippets/99", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSnippetDetail_DELETE(t *testing.T) {
	srv := newTestServer()
	sn := store.CreateSnippet(Snippet{Name: "S1", Command: "ls"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/api/snippets/%d", sn.Index), nil)
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	// Verify the snippet count decreased.
	snippets := store.ListSnippets()
	found := false
	for _, s := range snippets {
		if s.Index == sn.Index && s.Name == sn.Name {
			found = true
		}
	}
	if found {
		t.Fatal("snippet not deleted")
	}
}

func TestHandleSnippetDetail_DELETE_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/snippets/99", nil)
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSnippetDetail_InvalidIndex(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/snippets/abc", nil)
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSnippetDetail_EmptyPath(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/snippets/", nil)
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSnippetDetail_PUT_InvalidBody(t *testing.T) {
	srv := newTestServer()
	store.CreateSnippet(Snippet{Name: "Old", Command: "ls"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/snippets/0", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSnippetDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/snippets/0", nil)
	srv.handleSnippetDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
