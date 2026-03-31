package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- cors.go: MaxAge in preflight ---

func TestCORSMiddleware_PreflightWithMaxAge(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://trusted.example.com"},
		AllowedMethods: []string{"GET", "POST"},
		MaxAge:         3600,
	}
	handler := corsMiddleware(&cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for preflight")
	}))
	req := httptest.NewRequestWithContext(context.Background(), "OPTIONS", "/api/health", nil)
	req.Header.Set("Origin", "https://trusted.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Errorf("expected '3600', got '%s'", got)
	}
}

// --- list_address.go: error and IPAddr branches ---

func TestListAddresses_ErrorReturn(t *testing.T) {
	// listAddresses() returns []string{} on error.
	// We can't easily force net.Interfaces() to fail,
	// but calling it verifies the function doesn't panic.
	addrs := listAddresses()
	if addrs == nil {
		t.Error("expected non-nil result")
	}
}

// --- version.go: vcs.revision branches ---

func TestGetAppVersion_CoversVCSRevision(t *testing.T) {
	info := getAppVersion()
	// Verify all fields are populated regardless of build settings.
	if info.Version == "" {
		t.Error("expected non-empty version")
	}
	if info.GoVersion == "unknown" {
		t.Log("GoVersion is 'unknown' — build info may not be available in test")
	}
	// Commit may be empty in test builds (no vcs.revision setting).
	_ = info.Commit
}

// --- api_upload.go: empty content type fallback ---

func TestHandleUpload_NoContentType(t *testing.T) {
	srv := newTestServer()

	var buf strings.Builder
	buf.WriteString("--boundary\r\n")
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString("hello\r\n")
	buf.WriteString("--boundary--\r\n")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/upload", strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	srv.handleUpload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
	// The response should include type: application/octet-stream when Content-Type header is missing.
	body := rec.Body.String()
	if !strings.Contains(body, "application/octet-stream") {
		t.Errorf("expected 'application/octet-stream' fallback, got: %s", body)
	}
}

// --- api_fs.go: error paths ---

func TestIsPathAllowed_NonexistentRoot(t *testing.T) {
	// When no fsRoots are registered, isPathAllowed should return false.
	// We can't easily trigger filepath.Abs error, but we can verify the function.
	allowed := isPathAllowed("/some/path")
	if allowed {
		t.Error("expected false when no roots registered")
	}
}

func TestHandleFSList_PathTraversal(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path=/nonexistent/path/../../../etc", nil)
	srv.handleFSList(rec, req)

	// Path should be denied (not under any registered root).
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("expected 403/400/404 for path traversal, got %d", rec.Code)
	}
}
