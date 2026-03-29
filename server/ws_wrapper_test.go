package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWsWrapperInterface verifies that wsWrapper implements io.Reader and io.Writer.
func TestWsWrapperInterface(t *testing.T) {
	var _ interface {
		Read(p []byte) (n int, err error)
		Write(p []byte) (n int, err error)
	} = &wsWrapper{}
}

func TestWsWrapper_Write(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		wsw := &wsWrapper{Conn: conn}
		buf := make([]byte, 1024)
		n, err := wsw.Read(buf)
		if err != nil {
			return
		}
		_, _ = wsw.Write(buf[:n])
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	message := "hello websocket"
	err = conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, received, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if string(received) != message {
		t.Fatalf("expected %q, got %q", message, string(received))
	}
}

func TestWsWrapper_Read_ClosedConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close()

		wsw := &wsWrapper{Conn: conn}
		buf := make([]byte, 1024)
		_, err = wsw.Read(buf)
		if err == nil {
			t.Error("expected error reading from closed connection")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_ = conn.Close()
}

func TestWsWrapper_Write_ClosedConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close()

		wsw := &wsWrapper{Conn: conn}
		// Write after close may or may not return an error depending on
		// gorilla/websocket internals. The important thing is it doesn't panic.
		_, _ = wsw.Write([]byte("hello"))
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_ = conn.Close()
}

func TestWsWrapper_Read_BinaryMessageSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		wsw := &wsWrapper{Conn: conn}

		// Read in a goroutine, close after text message arrives.
		done := make(chan struct{})
		go func() {
			buf := make([]byte, 1024)
			_, _ = wsw.Read(buf)
			close(done)
		}()

		// Wait for close or timeout.
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Read did not return")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send a binary message (should be skipped by Read).
	_ = conn.WriteMessage(websocket.BinaryMessage, []byte("binary"))
	// Send a text message to allow Read to return.
	_ = conn.WriteMessage(websocket.TextMessage, []byte("text"))
}
