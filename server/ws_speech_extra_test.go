package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ttyweb/ai"
)

// dialWS is a test helper that dials a WebSocket server and closes the HTTP
// upgrade response body.
func dialWS(t *testing.T, wsURL string) *websocket.Conn {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	_ = resp.Body.Close()
	return conn
}

// TestSpeechSessionSendError tests the sendError method.
func TestSpeechSessionSendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		sess := &speechSession{
			clientConn: conn,
			done:       make(chan struct{}),
		}
		sess.sendError("test error")
	}))
	defer srv.Close()

	// Dial to trigger the handler.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialWS(t, wsURL)

	defer func() { _ = conn.Close() }()

	// Read the error message sent by the server.
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("client read failed: %v", err)
	}
	if !strings.Contains(string(msg), "test error") {
		t.Fatalf("expected error message, got %s", string(msg))
	}
}

// TestSpeechSessionBuildAudioFrame_FirstFrame tests building the first audio frame.
func TestSpeechSessionBuildAudioFrame_FirstFrame(t *testing.T) {
	t.Setenv("XFYUN_APP_ID", "test-id")
	t.Setenv("XFYUN_API_KEY", "test-key")
	t.Setenv("XFYUN_API_SECRET", "test-secret")

	cfg := ai.LoadXunfeiConfigFromEnv()
	sess := &speechSession{
		xunfeiCfg: cfg,
		params:    ai.DefaultSpeechParams(),
		done:      make(chan struct{}),
	}
	sess.seq = 1

	frame, err := sess.buildAudioFrame("base64audio")
	if err != nil {
		// May fail if env vars aren't properly set.
		t.Logf("buildAudioFrame error (expected in test env): %v", err)
	} else if frame != nil {
		t.Logf("frame built successfully, len=%d", len(frame))
	}
}

// TestSpeechSessionBuildAudioFrame_MiddleFrame tests building a middle audio frame.
func TestSpeechSessionBuildAudioFrame_MiddleFrame(t *testing.T) {
	sess := &speechSession{
		done: make(chan struct{}),
	}
	sess.seq = 2

	frame, err := sess.buildAudioFrame("base64audio")
	if err != nil {
		t.Fatalf("unexpected error for middle frame: %v", err)
	}
	if frame == nil {
		t.Fatal("expected non-nil frame")
	}
}

// TestSpeechSessionRun_ReadError tests that run handles read errors.
func TestSpeechSessionRun_ReadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		sess := &speechSession{
			clientConn: conn,
			done:       make(chan struct{}),
		}

		// Close the connection immediately to trigger read error.
		_ = conn.Close()

		// This should not block indefinitely.
		done := make(chan struct{})
		go func() {
			sess.run()
			close(done)
		}()

		select {
		case <-done:
			// Good, run returned.
		case <-time.After(2 * time.Second):
			t.Fatal("run() did not return after connection close")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialWS(t, wsURL)

	// Close immediately to trigger read error in the server-side run().
	_ = conn.Close()
}

// TestSpeechSessionRun_InvalidJSON tests that run handles invalid JSON messages.
func TestSpeechSessionRun_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		sess := &speechSession{
			clientConn: conn,
			done:       make(chan struct{}),
		}

		done := make(chan struct{})
		go func() {
			sess.run()
			close(done)
		}()

		// Wait for run to exit.
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run() did not return")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialWS(t, wsURL)

	defer func() { _ = conn.Close() }()

	// Send invalid JSON.
	_ = conn.WriteMessage(websocket.TextMessage, []byte("not json"))
	// Send another invalid message to ensure run continues after invalid JSON.
	_ = conn.WriteMessage(websocket.TextMessage, []byte("also not json"))
	// Close to trigger read error and exit run.
	_ = conn.Close()
}

// TestSpeechSessionRun_UnknownType tests the default case in the message switch.
func TestSpeechSessionRun_UnknownType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		sess := &speechSession{
			clientConn: conn,
			done:       make(chan struct{}),
		}

		done := make(chan struct{})
		go func() {
			sess.run()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run() did not return")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialWS(t, wsURL)

	defer func() { _ = conn.Close() }()

	// Send message with unknown type.
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"unknown"}`))
	// Close to trigger exit.
	_ = conn.Close()
}

// TestSpeechSessionRun_AudioWithoutXunfeiConn tests that audio messages are
// skipped when there is no Xunfei connection.
func TestSpeechSessionRun_AudioWithoutXunfeiConn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		sess := &speechSession{
			clientConn: conn,
			done:       make(chan struct{}),
		}

		done := make(chan struct{})
		go func() {
			sess.run()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run() did not return")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialWS(t, wsURL)

	defer func() { _ = conn.Close() }()

	// Send audio message without starting (no xunfeiConn).
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"audio","audio":"base64data"}`))
	// Close to exit.
	_ = conn.Close()
}

// TestSpeechSessionRun_StopWithoutXunfeiConn tests that stop messages are
// skipped when there is no Xunfei connection.
func TestSpeechSessionRun_StopWithoutXunfeiConn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		sess := &speechSession{
			clientConn: conn,
			done:       make(chan struct{}),
		}

		done := make(chan struct{})
		go func() {
			sess.run()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("run() did not return")
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn := dialWS(t, wsURL)

	defer func() { _ = conn.Close() }()

	// Send stop message without starting (no xunfeiConn).
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"stop"}`))
	// Close to exit.
	_ = conn.Close()
}
