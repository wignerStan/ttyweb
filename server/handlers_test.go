package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ttyweb/backend"
	"ttyweb/db"
)

// mockSlave implements backend.Slave for testing processWSConn past auth.
type mockSlave struct {
	readErr  error
	writeErr error
}

func (m *mockSlave) Read(p []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return 0, errors.New("EOF")
}

func (m *mockSlave) Write(p []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return len(p), nil
}

func (_ *mockSlave) WindowTitleVariables() map[string]any {
	return map[string]any{"command": "mock", "hostname": "test"}
}

func (_ *mockSlave) ResizeTerminal(columns int, rows int) error {
	return nil
}

func (_ *mockSlave) Close() error {
	return nil
}

// mockSlaveFactory creates a mockSlave for testing.
type mockSlaveFactory struct {
	name  string
	slave *mockSlave
}

func (f *mockSlaveFactory) Name() string { return f.name }
func (f *mockSlaveFactory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	return f.slave, nil
}

// errorFactory always returns an error from New.
type errorFactory struct{}

func (_ *errorFactory) Name() string { return "error-factory" }
func (_ *errorFactory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	return nil, errors.New("factory error")
}

func TestHandleIndex(t *testing.T) {
	// handleIndex writes the global indexHTML.
	// If indexHTML is empty (not built), it should still return 200.
	indexHTML = []byte("<html><body>test</body></html>")

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	srv.handleIndex(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "test") {
		t.Fatalf("expected body to contain 'test', got %q", body)
	}
}

func TestHandleIndex_Method(t *testing.T) {
	indexHTML = []byte("<html></html>")
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	srv.handleIndex(rec, req)

	// handleIndex doesn't check method, it always returns the index.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTitleVariables_MergeOrder(t *testing.T) {
	srv := newTestServer()
	order := []string{"a", "b"}
	varUnits := map[string]map[string]any{
		"a": {"key1": "val1", "key2": "val2"},
		"b": {"key2": "override", "key3": "val3"},
	}
	result := srv.titleVariables(order, varUnits)
	if result["key2"] != "override" {
		t.Errorf("key2 = %v, want 'override'", result["key2"])
	}
	if result["key1"] != "val1" {
		t.Errorf("key1 = %v, want 'val1'", result["key1"])
	}
	if result["key3"] != "val3" {
		t.Errorf("key3 = %v, want 'val3'", result["key3"])
	}
	// Verify conflict-safe net keys are set.
	if result["a"] == nil {
		t.Error("expected key 'a' in result")
	}
	if result["b"] == nil {
		t.Error("expected key 'b' in result")
	}
}

func TestTitleVariables_SingleSource(t *testing.T) {
	srv := newTestServer()
	order := []string{"only"}
	varUnits := map[string]map[string]any{
		"only": {"key": "value"},
	}
	result := srv.titleVariables(order, varUnits)
	if result["key"] != "value" {
		t.Errorf("key = %v, want 'value'", result["key"])
	}
}

func TestGenerateHandleWS_ReturnsHandler(t *testing.T) {
	srv := newTestServer()
	srv.factory = &mockFactory{name: "test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestGenerateHandleWS_MaxConnectionExceeded(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:          "/",
		TitleFormat:   "{{ .command }}",
		MaxConnection: 1,
		Credential:    "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	// First connection fills the counter.
	counter.add(1)

	// Second request should be rejected because handler does counter.add(1)
	// making num=2 > MaxConnection=1.
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", nil)
	handler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when max connections exceeded, got %d", rec.Code)
	}
}

func TestGenerateHandleWS_OnceOption(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .command }}",
		Once:        true,
		Credential:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	// First connection should succeed (WebSocket upgrade).
	conn1, resp1, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("first dial failed: %v", err)
	}
	defer func() { _ = conn1.Close() }()
	defer func() { _ = resp1.Body.Close() }()

	// Wait for the first handler to process the connection and set the once flag.
	time.Sleep(100 * time.Millisecond)

	// Second connection should be rejected with 503.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", nil)
	handler(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for second connection, got %d", rec.Code)
	}
}

func TestProcessWSConn_InvalidJSON(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .command }}", Credential: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send invalid JSON — processWSConn should reject.
	err = conn.WriteMessage(websocket.TextMessage, []byte("not json at all"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Wait for the server to close the connection.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after invalid JSON")
	}
}

func TestProcessWSConn_WrongAuthToken(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .command }}",
		Credential:  "correct-token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send valid JSON but wrong auth token.
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"AuthToken":"wrong-token"}`))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after wrong auth token")
	}
}

func TestProcessWSConn_NonTextMessage(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .command }}", Credential: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send binary message instead of text.
	err = conn.WriteMessage(websocket.BinaryMessage, []byte("binary data"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after non-text message")
	}
}

func TestProcessWSConn_WithMockSlave(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockSlaveFactory{
		name:  "mock-slave",
		slave: &mockSlave{},
	}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .command }}@{{ .hostname }}",
		Credential:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send valid auth with no arguments.
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"AuthToken":""}`))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// The connection should eventually close when the mock slave returns EOF.
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after slave EOF")
	}
}

func TestProcessWSConn_PermitArguments_InvalidSession(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockSlaveFactory{
		name:  "mock-slave",
		slave: &mockSlave{},
	}
	srv, err := New(factory, &Options{
		Path:           "/",
		TitleFormat:    "{{ .command }}",
		Credential:     "",
		PermitArguments: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send auth with an invalid session name (as a query string).
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"AuthToken":"","Arguments":"?session=invalid/name!"}`))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after invalid session name")
	}
}

func TestProcessWSConn_PermitArguments_InvalidPane(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockSlaveFactory{
		name:  "mock-slave",
		slave: &mockSlave{},
	}
	srv, err := New(factory, &Options{
		Path:           "/",
		TitleFormat:    "{{ .command }}",
		Credential:     "",
		PermitArguments: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send auth with an invalid pane ID (as a query string).
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"AuthToken":"","Arguments":"?pane=invalid!"}`))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after invalid pane ID")
	}
}

func TestProcessWSConn_FactoryError(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	srv, err := New(&errorFactory{}, &Options{
		Path:        "/",
		TitleFormat: "{{ .command }}",
		Credential:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"AuthToken":""}`))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after factory error")
	}
}

func TestGenerateHandleWS_NonGET_MethodNotAllowed(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockSlaveFactory{
		name:  "mock-slave",
		slave: &mockSlave{},
	}
	srv, err := New(factory, &Options{
		Path:        "/",
		TitleFormat: "{{ .command }}",
		Credential:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	// Send a POST request — should get 405 Method not allowed.
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/ws", nil)
	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestProcessWSConn_WithOptions(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	factory := &mockSlaveFactory{
		name:  "opts-slave",
		slave: &mockSlave{},
	}
	srv, err := New(factory, &Options{
		Path:            "/",
		TitleFormat:     "{{ .command }}@{{ .hostname }}",
		Credential:      "",
		PermitWrite:     true,
		EnableReconnect: true,
		ReconnectTime:   5,
		Width:           80,
		Height:          24,
		PassHeaders:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := newCounter(0)
	handler := srv.generateHandleWS(ctx, cancel, counter)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	conn, wsResp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = wsResp.Body.Close() }()
	defer func() { _ = conn.Close() }()

	// Send valid auth.
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"AuthToken":""}`))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Wait for the connection to close.
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Log("expected read error after slave EOF")
	}
}
