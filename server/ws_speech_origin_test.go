package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestSpeechUpgrader_SameOriginAllowed verifies that the speech WebSocket
// upgrader accepts connections where Origin matches the request Host.
func TestSpeechUpgrader_SameOriginAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := speechUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("same-origin dial should succeed, got error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = resp.Body.Close() }()
}

// TestSpeechUpgrader_CrossOriginRejected verifies that the speech WebSocket
// upgrader rejects connections where Origin does not match the request Host.
func TestSpeechUpgrader_CrossOriginRejected(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := speechUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	header := http.Header{}
	header.Set("Origin", "http://evil.example.com")

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("cross-origin dial should be rejected, but it succeeded")
	}
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for cross-origin, got %d", resp.StatusCode)
		}
	}
}

// TestSpeechUpgrader_EmptyOriginAllowed verifies that the speech WebSocket
// upgrader accepts connections with no Origin header (non-browser clients).
func TestSpeechUpgrader_EmptyOriginAllowed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := speechUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	dialer := &websocket.Dialer{
		HandshakeTimeout: 3 * time.Second,
	}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("empty-origin dial should succeed, got error: %v", err)
	}
	defer func() { _ = conn.Close() }()
	defer func() { _ = resp.Body.Close() }()
}
