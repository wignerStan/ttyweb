package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates a minimal Server for testing middleware methods.
// It resets the global MemoryStore to ensure test isolation.
func newTestServer() *Server {
	store = NewMemoryStore()
	return &Server{
		options: &Options{Path: "/"},
	}
}

func TestWrapHeaders_SetsSecurityHeaders(t *testing.T) {
	srv := newTestServer()
	handler := srv.wrapHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Permissions-Policy":     "camera=(), geolocation=()",
	}

	for name, expected := range headers {
		got := rec.Header().Get(name)
		if got != expected {
			t.Errorf("Header %q: expected %q, got %q", name, expected, got)
		}
	}

	// Server header includes dynamic version, verify prefix
	server := rec.Header().Get("Server")
	if server == "" {
		t.Fatal(`Header "Server" is empty`)
	}
	if server[:6] != "ttyweb" {
		t.Errorf(`Header "Server": expected "ttyweb/..." prefix, got %q`, server)
	}
}

func TestWrapBasicAuth_ValidCredentials(t *testing.T) {
	srv := newTestServer()
	credential := "user:pass"
	handler := srv.wrapBasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), credential)

	auth := base64.StdEncoding.EncodeToString([]byte(credential))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+auth)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
}

func TestWrapBasicAuth_InvalidCredentials(t *testing.T) {
	srv := newTestServer()
	credential := "user:pass"
	handler := srv.wrapBasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), credential)

	auth := base64.StdEncoding.EncodeToString([]byte("wrong:creds"))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+auth)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Code)
	}
}

func TestWrapBasicAuth_MissingHeader(t *testing.T) {
	srv := newTestServer()
	credential := "user:pass"
	handler := srv.wrapBasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), credential)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Code)
	}

	// Verify WWW-Authenticate header is set
	authHeader := rec.Header().Get("WWW-Authenticate")
	if authHeader != `Basic realm="GoTTY"` {
		t.Fatalf("Expected WWW-Authenticate header, got %q", authHeader)
	}
}

func TestWrapBasicAuth_MalformedHeader(t *testing.T) {
	srv := newTestServer()
	credential := "user:pass"
	handler := srv.wrapBasicAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), credential)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic not-valid-base64!!!")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Code)
	}
}

func TestWrapLogger_LogsRequest(t *testing.T) {
	srv := newTestServer()
	called := false
	handler := srv.wrapLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test-path", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("Inner handler was not called by wrapLogger")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Code)
	}
}
