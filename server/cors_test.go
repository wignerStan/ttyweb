package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_DefaultDenyAll(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no Access-Control-Allow-Origin header")
	}
}

func TestCORSMiddleware_SameOriginAllowed(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("same-origin request should pass, got %d", rec.Code)
	}
}

func TestCORSMiddleware_ConfiguredOrigin(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://trusted.example.com"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
	}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.Header.Set("Origin", "https://trusted.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://trusted.example.com" {
		t.Errorf("expected 'https://trusted.example.com', got '%s'", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("expected 'Vary: Origin', got 'Vary: %s'", got)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://trusted.example.com"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
	}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for preflight")
	}))
	req := httptest.NewRequestWithContext(context.Background(), "OPTIONS", "/api/health", nil)
	req.Header.Set("Origin", "https://trusted.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}

func TestCORSMiddleware_UntrustedOriginBlocked(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{AllowedOrigins: []string{"https://trusted.example.com"}}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.Header.Set("Origin", "https://untrusted.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("untrusted origin should not get CORS headers")
	}
}

func TestCORSMiddleware_WithCredentials(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://trusted.example.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Custom-Header"},
		MaxAge:           3600,
	}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.Header.Set("Origin", "https://trusted.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected 'true', got '%s'", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Custom-Header" {
		t.Errorf("expected 'X-Custom-Header', got '%s'", got)
	}
}
