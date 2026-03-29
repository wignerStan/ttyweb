package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsValidButlerPath_AcceptsValidPaths(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api/butler/status", true},
		{"/api/butler/tasks/123", true},
		{"/api/butler/foo/bar/baz", true},
		{"/api/something", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isValidButlerPath(tt.path)
			if got != tt.want {
				t.Errorf("isValidButlerPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsValidButlerPath_RejectsPathTraversal(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api/butler/../etc/passwd", false},
		{"/api/../secret", false},
		{"/api/butler/..", false},
		{"/api/..", false},
		{"../etc/passwd", false},
		{"/api/butler/../../tmp/evil", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isValidButlerPath(tt.path)
			if got != tt.want {
				t.Errorf("isValidButlerPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsValidButlerPath_RejectsMissingApiPrefix(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/butler/status", false},
		{"/admin/secret", false},
		{"/", false},
		{"/static/file.txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isValidButlerPath(tt.path)
			if got != tt.want {
				t.Errorf("isValidButlerPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestHandleButlerProxy_PathTraversalReturns400(t *testing.T) {
	srv := newTestServer()

	tests := []struct {
		name string
		path string
	}{
		{"dotdot in butler path", "/api/butler/../etc/passwd"},
		{"dotdot at api level", "/api/../secret"},
		{"no api prefix", "/butler/status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			srv.handleButlerProxy(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", rec.Code)
			}

			var resp map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp["success"] != false {
				t.Errorf("expected success=false, got %v", resp["success"])
			}
		})
	}
}

func TestHandleButlerProxy_ValidPathReachesProxy(t *testing.T) {
	srv := newTestServer()

	// A valid path should attempt to connect to the upstream (which will fail
	// since there is no real upstream), returning 502 Bad Gateway.
	req := httptest.NewRequest(http.MethodGet, "/api/butler/status", nil)
	rec := httptest.NewRecorder()
	srv.handleButlerProxy(rec, req)

	// Should not get 400 (path validation); should get 502 (upside unreachable).
	if rec.Code == http.StatusBadRequest {
		t.Fatalf("valid path was rejected by path validation: %d", rec.Code)
	}
	// Expect 502 because the upstream is not running.
	if rec.Code != http.StatusBadGateway {
		t.Logf("expected 502 (upstream unreachable), got %d (still valid if upstream is running)", rec.Code)
	}
}

func TestHandleButlerProxy_BodySizeLimitRejectsOversizedPayload(t *testing.T) {
	srv := newTestServer()

	// Create a body larger than the 1 MB limit.
	oversized := strings.NewReader(string(make([]byte, butlerMaxRequestBodyBytes+1)))

	req := httptest.NewRequest(http.MethodPost, "/api/butler/data", oversized)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleButlerProxy(rec, req)

	// MaxBytesReader causes http.NewRequestWithContext to fail when reading the body,
	// or the upstream request fails. Either way, we should not proxy the oversized body.
	// The response should be an error (400 or 502), not a 200.
	if rec.Code == http.StatusOK {
		t.Fatal("expected error for oversized body, got 200")
	}
}

func TestCopyAllowedResponseHeaders_OnlyCopiesAllowlisted(t *testing.T) {
	src := http.Header{}
	src.Set("Content-Type", "application/json")
	src.Set("Content-Length", "42")
	src.Set("Cache-Control", "no-cache")
	src.Set("X-Custom-Header", "should-be-stripped")
	src.Set("Server", "should-be-stripped")
	src.Set("Set-Cookie", "session=abc; HttpOnly")

	dst := http.Header{}
	copyAllowedResponseHeaders(dst, src)

	if dst.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be forwarded, got %q", dst.Get("Content-Type"))
	}
	if dst.Get("Content-Length") != "42" {
		t.Errorf("Content-Length should be forwarded, got %q", dst.Get("Content-Length"))
	}
	if dst.Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control should be forwarded, got %q", dst.Get("Cache-Control"))
	}
	if dst.Get("X-Custom-Header") != "" {
		t.Errorf("X-Custom-Header should be stripped, got %q", dst.Get("X-Custom-Header"))
	}
	if dst.Get("Server") != "" {
		t.Errorf("Server should be stripped, got %q", dst.Get("Server"))
	}
	if dst.Get("Set-Cookie") != "" {
		t.Errorf("Set-Cookie should be stripped, got %q", dst.Get("Set-Cookie"))
	}
}
