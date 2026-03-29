package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleIndex(t *testing.T) {
	// handleIndex writes the global indexHTML.
	// If indexHTML is empty (not built), it should still return 200.
	indexHTML = []byte("<html><body>test</body></html>")

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	srv.handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "test") {
		t.Fatalf("expected body to contain 'test', got %q", body)
	}
}

func TestHandleIndex_Method(t *testing.T) {
	indexHTML = []byte("<html></html>")
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	srv.handleIndex(rec, req)

	// handleIndex doesn't check method, it always returns the index.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
