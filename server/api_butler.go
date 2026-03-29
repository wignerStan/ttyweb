package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// butlerProxyClient is a shared HTTP client for non-SSE Butler proxy requests.
// Reusing the client preserves idle connections across requests.
var butlerProxyClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
	},
	Timeout: 60 * time.Second,
}

// hopByHopHeaders are headers that should not be forwarded to the upstream.
var hopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailers":            true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

// butlerAllowedResponseHeaders is an allowlist of response headers forwarded
// from the upstream Butler service. All other headers are stripped.
var butlerAllowedResponseHeaders = map[string]bool{
	"Content-Type":   true,
	"Content-Length": true,
	"Cache-Control":  true,
	"Last-Modified":  true,
	"Etag":           true,
	"X-Request-Id":   true,
}

// butlerMaxRequestBodyBytes is the maximum allowed request body size for
// normal (non-SSE) Butler proxy requests (1 MB).
const butlerMaxRequestBodyBytes int64 = 1 << 20

// butlerTarget resolves the Butler service address from environment variables
// with a default fallback of "localhost:9999".
func butlerTarget() string {
	host := os.Getenv("BUTLER_HOST")
	port := os.Getenv("BUTLER_PORT")
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "9999"
	}
	return net.JoinHostPort(host, port)
}

// isSSERequest checks whether the request is asking for an SSE stream.
func isSSERequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// isValidButlerPath validates that the request path does not contain path
// traversal sequences and starts with the expected /api/ prefix.
func isValidButlerPath(path string) bool {
	if !strings.HasPrefix(path, "/api/") {
		return false
	}
	return !strings.Contains(path, "..")
}

// copyHeaders copies headers from src to dst, skipping hop-by-hop headers.
func copyHeaders(dst, src http.Header) {
	for key, vals := range src {
		if hopByHopHeaders[key] {
			continue
		}
		for _, val := range vals {
			dst.Add(key, val)
		}
	}
}

// copyAllowedResponseHeaders copies only allowlisted headers from src to dst.
func copyAllowedResponseHeaders(dst, src http.Header) {
	for key, vals := range src {
		if !butlerAllowedResponseHeaders[key] {
			continue
		}
		for _, val := range vals {
			dst.Add(key, val)
		}
	}
}

// handleButlerProxy is a real HTTP reverse proxy for the Butler orchestration service.
// All requests to /api/butler/* are forwarded to the upstream Butler service.
// SSE requests are detected and handled via direct byte piping with no buffering.
func (server *Server) handleButlerProxy(w http.ResponseWriter, r *http.Request) {
	if !isValidButlerPath(r.URL.Path) {
		log.Printf("[Butler Proxy] rejected invalid path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		writeAPIError(w, http.StatusBadRequest, "invalid request path")
		return
	}

	target := butlerTarget()

	// Forward the full request path and query string to the upstream Butler service.
	targetURL := fmt.Sprintf("http://%s%s", target, r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	if isSSERequest(r) {
		server.proxySSE(w, r, targetURL, target)
		return
	}

	server.proxyNormal(w, r, targetURL, target)
}

// proxySSE handles SSE (Server-Sent Events) requests by hijacking the connection
// and piping bytes directly without buffering. No timeout is applied so that
// long-lived SSE streams are not interrupted. The request context is propagated
// so that client disconnections cancel the upstream request.
func (_ *Server) proxySSE(w http.ResponseWriter, r *http.Request, targetURL, target string) {
	log.Printf("[Butler Proxy] SSE request: %s %s -> %s", r.Method, r.URL.Path, targetURL)

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		log.Printf("[Butler Proxy] SSE: failed to create request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		writeAPIError(w, http.StatusBadGateway, "Butler service unavailable")
		return
	}

	proxyReq.Header.Set("Accept", "text/event-stream")
	proxyReq.Header.Set("Host", target)
	if r.Header.Get("Content-Type") != "" {
		proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	}

	client := &http.Client{} // no timeout for SSE
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("[Butler Proxy] SSE: upstream request failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		writeAPIError(w, http.StatusBadGateway, "Butler service unavailable")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Flush headers immediately to start the SSE stream.
	flusher, canFlush := w.(http.Flusher)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(resp.StatusCode)
	if canFlush {
		flusher.Flush()
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				log.Printf("[Butler Proxy] SSE: write to client failed: %v", writeErr)
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				log.Printf("[Butler Proxy] SSE: upstream read ended: %v", readErr)
			}
			return
		}
	}
}

// proxyNormal handles standard (non-SSE) proxy requests with a 30s dial timeout,
// 60s request timeout, and a 1 MB body size limit. The request context is
// propagated so that client disconnections cancel the upstream request.
// Response headers are filtered through an allowlist.
func (_ *Server) proxyNormal(w http.ResponseWriter, r *http.Request, targetURL, target string) {
	log.Printf("[Butler Proxy] %s %s -> %s", r.Method, r.URL.Path, targetURL)

	limitedBody := http.MaxBytesReader(w, r.Body, butlerMaxRequestBodyBytes)

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, limitedBody)
	if err != nil {
		log.Printf("[Butler Proxy] failed to create request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		writeAPIError(w, http.StatusBadGateway, "Butler service unavailable")
		return
	}

	// Forward headers, stripping hop-by-hop.
	copyHeaders(proxyReq.Header, r.Header)
	proxyReq.Header.Set("Host", target)

	resp, err := butlerProxyClient.Do(proxyReq)
	if err != nil {
		log.Printf("[Butler Proxy] upstream request failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		writeAPIError(w, http.StatusBadGateway, "Butler service unavailable")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Copy only allowlisted response headers.
	copyAllowedResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("[Butler Proxy] error copying response body: %v", err)
	}
}
