package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddleware_GETPasses(t *testing.T) {
	t.Parallel()
	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET should pass through, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTWithoutHeader(t *testing.T) {
	t.Parallel()
	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTWithHeader(t *testing.T) {
	t.Parallel()
	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/sessions", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("POST with X-Requested-With should pass, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_DELETEWithoutHeader(t *testing.T) {
	t.Parallel()
	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))
	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/api/sessions/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_OPTIONSExempt(t *testing.T) {
	t.Parallel()
	handler := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "OPTIONS", "/api/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS should be exempt, got %d", rec.Code)
	}
}
