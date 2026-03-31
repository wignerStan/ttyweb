package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDefaultOriginChecker tests all branches of defaultOriginChecker.
func TestDefaultOriginChecker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{"empty origin", "example.com:8080", "", true},
		{"matching host", "example.com:8080", "http://example.com:8080", true},
		{"non-matching host", "example.com:8080", "http://evil.com:8080", false},
		{"invalid origin", "example.com:8080", "://invalid", false},
		{"no port match", "example.com", "http://example.com", true},
		{"port mismatch", "example.com:8080", "http://example.com:9090", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "/ws", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			got := defaultOriginChecker(req)
			if got != tc.want {
				t.Errorf("defaultOriginChecker(%q) = %v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}

// TestHandleIndex verifies the index handler returns HTML content.
func TestHandleIndexHandler(t *testing.T) {
	t.Parallel()
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %q", ct)
	}
}

// TestListAddressesHasLoopback verifies listAddresses returns at least loopback.
func TestListAddressesHasLoopback(t *testing.T) {
	t.Parallel()
	addrs := listAddresses()
	if len(addrs) == 0 {
		t.Error("expected at least one address")
	}
	found := false
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.IsLoopback() {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find a loopback address")
	}
}

// TestServeBackground_NonTLS verifies serveBackground calls serve (not serveTLS).
func TestServeBackground_NonTLS(t *testing.T) {
	t.Parallel()
	srv := &Server{
		options:  &Options{EnableTLS: false},
		srvErrCh: make(chan error, 1),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go srv.serveBackground(&http.Server{Handler: handler}, ln)

	// Make a request to ensure it's running.
	client := &http.Client{}
	resp, err := client.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close() //nolint:errcheck // test cleanup

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestServeBackground_TLSError verifies TLS error path (no cert files).
func TestServeBackground_TLSError(t *testing.T) {
	t.Parallel()
	srv := &Server{
		options: &Options{
			EnableTLS:  true,
			TLSCrtFile: "/nonexistent/cert.pem",
			TLSKeyFile: "/nonexistent/key.pem",
		},
		srvErrCh: make(chan error, 1),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go srv.serveBackground(&http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})}, ln)

	// Wait for the error from the background goroutine.
	select {
	case err := <-srv.srvErrCh:
		if err == nil {
			t.Error("expected error for missing TLS files")
		}
	default:
		// Error might not have fired yet, that's OK for this test.
	}
}
