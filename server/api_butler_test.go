package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// butlerTarget()
// ---------------------------------------------------------------------------

func TestButlerTarget_CustomEnvVars(t *testing.T) {
	t.Setenv("BUTLER_HOST", "butler.internal")
	t.Setenv("BUTLER_PORT", "3000")

	got := butlerTarget()
	want := "butler.internal:3000"
	if got != want {
		t.Errorf("butlerTarget() = %q, want %q", got, want)
	}
}

func TestButlerTarget_Defaults(t *testing.T) {
	// Ensure env vars are cleared so defaults are used.
	t.Setenv("BUTLER_HOST", "")
	t.Setenv("BUTLER_PORT", "")

	got := butlerTarget()
	want := "localhost:9999"
	if got != want {
		t.Errorf("butlerTarget() = %q, want %q", got, want)
	}
}

func TestButlerTarget_HostOnly(t *testing.T) {
	t.Setenv("BUTLER_HOST", "10.0.0.1")
	t.Setenv("BUTLER_PORT", "")

	got := butlerTarget()
	want := "10.0.0.1:9999"
	if got != want {
		t.Errorf("butlerTarget() = %q, want %q", got, want)
	}
}

func TestButlerTarget_PortOnly(t *testing.T) {
	t.Setenv("BUTLER_HOST", "")
	t.Setenv("BUTLER_PORT", "8080")

	got := butlerTarget()
	want := "localhost:8080"
	if got != want {
		t.Errorf("butlerTarget() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// isSSERequest()
// ---------------------------------------------------------------------------

func TestIsSSERequest_True(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/stream", nil)
	req.Header.Set("Accept", "text/event-stream")

	if !isSSERequest(req) {
		t.Error("isSSERequest() should return true for Accept: text/event-stream")
	}
}

func TestIsSSERequest_False(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/data", nil)
	req.Header.Set("Accept", "application/json")

	if isSSERequest(req) {
		t.Error("isSSERequest() should return false for Accept: application/json")
	}
}

func TestIsSSERequest_EmptyAccept(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/data", nil)

	if isSSERequest(req) {
		t.Error("isSSERequest() should return false when Accept header is empty")
	}
}

func TestIsSSERequest_MultipleAcceptValues(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/stream", nil)
	req.Header.Set("Accept", "application/json, text/event-stream")

	if !isSSERequest(req) {
		t.Error("isSSERequest() should return true when text/event-stream is among multiple Accept values")
	}
}

// ---------------------------------------------------------------------------
// copyHeaders()
// ---------------------------------------------------------------------------

func TestCopyHeaders_SkipsHopByHop(t *testing.T) {
	src := http.Header{}
	src.Set("Content-Type", "application/json")
	src.Set("X-Custom", "value")
	src.Set("Connection", "keep-alive")
	src.Set("Transfer-Encoding", "chunked")

	dst := http.Header{}
	copyHeaders(dst, src)

	if dst.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type not copied: got %q", dst.Get("Content-Type"))
	}
	if dst.Get("X-Custom") != "value" {
		t.Errorf("X-Custom not copied: got %q", dst.Get("X-Custom"))
	}
	if dst.Get("Connection") != "" {
		t.Errorf("Connection (hop-by-hop) should not be copied, got %q", dst.Get("Connection"))
	}
	if dst.Get("Transfer-Encoding") != "" {
		t.Errorf("Transfer-Encoding (hop-by-hop) should not be copied, got %q", dst.Get("Transfer-Encoding"))
	}
}

func TestCopyHeaders_MultiValues(t *testing.T) {
	src := http.Header{}
	src.Add("Set-Cookie", "a=1")
	src.Add("Set-Cookie", "b=2")

	dst := http.Header{}
	copyHeaders(dst, src)

	values := dst.Values("Set-Cookie")
	if len(values) != 2 || values[0] != "a=1" || values[1] != "b=2" {
		t.Errorf("expected two Set-Cookie values, got %v", values)
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — normal (non-SSE) proxy via httptest upstream
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_NormalGET(t *testing.T) {
	var expectedHost string
	// Start a mock upstream Butler server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/butler/status" {
			t.Errorf("upstream received path %q, want /api/butler/status", r.URL.Path)
		}
		// Verify Host header is set to the target.
		if r.Host != expectedHost {
			t.Errorf("upstream Host = %q, want %q", r.Host, expectedHost)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer upstream.Close()

	expectedHost = strings.TrimPrefix(upstream.URL, "http://")

	// Point the proxy at the mock upstream.
	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/status", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("response body status = %v, want ok", body["status"])
	}
}

func TestHandleButlerProxy_NormalPOST(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("upstream Method = %q, want POST", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"key":"value"}` {
			t.Errorf("upstream received body %q, want {\"key\":\"value\"}", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"created": true,
		})
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/butler/resources", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["created"] != true {
		t.Errorf("response body created = %v, want true", body["created"])
	}
}

func TestHandleButlerProxy_ForwardsQueryParams(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.RawQuery
		if query != "foo=bar&baz=1" {
			t.Errorf("upstream query = %q, want foo=bar&baz=1", query)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/search?foo=bar&baz=1", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestHandleButlerProxy_NormalUpstreamError(t *testing.T) {
	// Point at a port that nothing is listening on to trigger a connection error.
	t.Setenv("BUTLER_HOST", "127.0.0.1")
	t.Setenv("BUTLER_PORT", "1")

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/status", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}

	var body apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Success != false {
		t.Error("expected success=false on upstream error")
	}
	if !strings.Contains(body.Error, "Butler service unavailable") {
		t.Errorf("error message = %q, want to contain 'Butler service unavailable'", body.Error)
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — SSE proxy via httptest upstream
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_SSEStream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "text/event-stream" {
			t.Errorf("upstream Accept = %q, want text/event-stream", accept)
		}

		flusher, canFlush := w.(http.Flusher)
		if !canFlush {
			t.Fatal("upstream response writer does not support flushing")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		// Send a few SSE events.
		for _, event := range []string{
			"data: hello\n\n",
			"data: world\n\n",
		} {
			_, _ = w.Write([]byte(event))
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/stream", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Verify SSE response headers.
	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", contentType)
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("X-Accel-Buffering") != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", rec.Header().Get("X-Accel-Buffering"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, "data: hello") {
		t.Errorf("response body missing 'data: hello', got: %q", body)
	}
	if !strings.Contains(body, "data: world") {
		t.Errorf("response body missing 'data: world', got: %q", body)
	}
}

func TestHandleButlerProxy_SSEUpstreamError(t *testing.T) {
	// Point at a port that nothing is listening on.
	t.Setenv("BUTLER_HOST", "127.0.0.1")
	t.Setenv("BUTLER_PORT", "1")

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/stream", nil)
	req.Header.Set("Accept", "text/event-stream")
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}

	var body apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.Success != false {
		t.Error("expected success=false on SSE upstream error")
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — path with .. (no path validation exists, so verify
// the path is forwarded as-is to upstream, which is responsible for sanitizing).
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_PathTraversal(t *testing.T) {
	// The hardened proxy rejects path traversal before reaching upstream.
	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/../../../etc/passwd", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — hop-by-hop headers are stripped in normal proxy
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_HopByHopHeadersStripped(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that hop-by-hop headers were NOT forwarded.
		if r.Header.Get("Connection") != "" {
			t.Errorf("Connection header should be stripped, got %q", r.Header.Get("Connection"))
		}
		if r.Header.Get("Transfer-Encoding") != "" {
			t.Errorf("Transfer-Encoding header should be stripped, got %q", r.Header.Get("Transfer-Encoding"))
		}
		if r.Header.Get("Keep-Alive") != "" {
			t.Errorf("Keep-Alive header should be stripped, got %q", r.Header.Get("Keep-Alive"))
		}

		w.Header().Set("X-Request-Id", "upstream-123")
		w.Header().Set("Connection", "close") // should be stripped on response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/test", nil)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Transfer-Encoding", "chunked")
	req.Header.Set("Keep-Alive", "timeout=5")
	req.Header.Set("X-Custom", "forward-me")
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Verify hop-by-hop headers were stripped from the response.
	if rec.Header().Get("Connection") != "" {
		t.Errorf("response Connection header should be stripped, got %q", rec.Header().Get("Connection"))
	}
	// Allowlisted upstream header should be forwarded.
	if rec.Header().Get("X-Request-Id") != "upstream-123" {
		t.Errorf("X-Request-Id header missing from response, got %q", rec.Header().Get("X-Request-Id"))
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — normal proxy forwards custom request headers
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_ForwardsCustomHeaders(t *testing.T) {
	var receivedAuth string
	var receivedXRequestID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedXRequestID = r.Header.Get("X-Request-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Request-Id", "req-123")
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", receivedAuth, "Bearer test-token")
	}
	if receivedXRequestID != "req-123" {
		t.Errorf("X-Request-Id header = %q, want %q", receivedXRequestID, "req-123")
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — upstream non-200 status codes are forwarded
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_UpstreamNotFound(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "not found",
		})
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/nonexistent", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["error"] != "not found" {
		t.Errorf("response body error = %v, want 'not found'", body["error"])
	}
}

func TestHandleButlerProxy_UpstreamInternalServerError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/error", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — SSE proxy forwards Content-Type when set
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_SSEForwardsContentType(t *testing.T) {
	var receivedContentType string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: done\n\n"))
	}))
	defer upstream.Close()

	host, port := parseHostPort(upstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/butler/stream", strings.NewReader(`{"cmd":"run"}`))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if receivedContentType != "application/json" {
		t.Errorf("SSE upstream Content-Type = %q, want application/json", receivedContentType)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// handleButlerProxy — upstream timeout scenario
// ---------------------------------------------------------------------------

func TestHandleButlerProxy_UpstreamTimeout(t *testing.T) {
	// Use a short-timeout client to simulate timeout behavior.
	slowUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowUpstream.Close()

	// Override butlerProxyClient with a short timeout for this test.
	originalClient := butlerProxyClient
	butlerProxyClient = &http.Client{Timeout: 100 * time.Millisecond}
	defer func() { butlerProxyClient = originalClient }()

	host, port := parseHostPort(slowUpstream.URL)
	t.Setenv("BUTLER_HOST", host)
	t.Setenv("BUTLER_PORT", port)

	srv := newTestServer()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/slow", nil)
	rec := httptest.NewRecorder()

	srv.handleButlerProxy(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502 on timeout, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseHostPort extracts the host and port from an httptest.Server URL.
// E.g. "http://127.0.0.1:12345" -> "127.0.0.1", "12345"
func parseHostPort(rawURL string) (string, string) {
	addr := strings.TrimPrefix(rawURL, "http://")
	// Find the last colon (handles IPv6 if needed, though httptest uses IPv4).
	idx := strings.LastIndex(addr, ":")
	if idx == -1 {
		return addr, ""
	}
	return addr[:idx], addr[idx+1:]
}
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
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tt.path, nil)
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
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/butler/status", nil)
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

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/butler/data", oversized)
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
