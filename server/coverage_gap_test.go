package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// --- api_fs.go coverage ---

func TestHandleFSList_NotFound(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	registerFSRoot("/tmp")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fs?path=/tmp/nonexistent_dir_xyz", nil)
	srv.handleFSList(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleFSList_Forbidden(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	fsRootsMu.Lock()
	orig := fsRoots
	fsRoots = []string{}
	fsRootsMu.Unlock()
	defer func() {
		fsRootsMu.Lock()
		fsRoots = orig
		fsRootsMu.Unlock()
	}()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fs?path=/tmp", nil)
	srv.handleFSList(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestHandleFSList_PostMethod(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/fs", nil)
	srv.handleFSList(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleFSList_Success(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	dir := t.TempDir()
	registerFSRoot(dir)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fs?path="+dir, nil)
	srv.handleFSList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool  `json:"success"`
		Data    []any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

// --- event_bus.go coverage ---

func TestSSEHandler_BusClosedDuringStream(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	handler := NewSSEHandler(bus)

	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/tasks/events/stream", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(rec, req)
	}()

	time.Sleep(20 * time.Millisecond)
	bus.Close()

	select {
	case <-done:
		// correct: handler exited
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not exit after bus close")
	}
	cancel()
}

// --- handlers.go coverage ---

func TestClassifyWSCloseError_Nil(t *testing.T) {
	result := classifyWSCloseError(nil, "test-backend")
	if result != "normal close" {
		t.Errorf("expected 'normal close', got %q", result)
	}
}

func TestClassifyWSCloseError_Canceled(t *testing.T) {
	result := classifyWSCloseError(context.Canceled, "test-backend")
	if result != "cancelation" {
		t.Errorf("expected 'cancelation', got %q", result)
	}
}

func TestClassifyWSCloseError_Default(t *testing.T) {
	result := classifyWSCloseError(os.ErrClosed, "test-backend")
	if !strings.Contains(result, "an error:") {
		t.Errorf("expected 'an error:' prefix, got %q", result)
	}
}

func TestWebttyOptions_Basic(t *testing.T) {
	srv := &Server{
		options: &Options{PermitWrite: true},
	}
	opts := srv.webttyOptions([]byte("test-title"))
	if len(opts) == 0 {
		t.Error("expected at least one option")
	}
}

func TestWebttyOptions_AllFlags(t *testing.T) {
	srv := &Server{
		options: &Options{
			PermitWrite:     true,
			EnableReconnect: true,
			ReconnectTime:   5,
			Width:           120,
			Height:          40,
		},
	}
	opts := srv.webttyOptions([]byte("title"))
	if len(opts) < 4 {
		t.Errorf("expected at least 4 options with all flags set, got %d", len(opts))
	}
}

// --- ws_ai_stream.go coverage ---

func TestResolveSystemPrompt_ExistingRole(t *testing.T) {
	prompt := resolveSystemPrompt("cli")
	if prompt == "" {
		t.Error("expected non-empty system prompt")
	}
}

// --- api_summary.go coverage ---

func TestHandleGetSummary_MethodNotAllowed_Gap(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()
	srv := newTestServerWithSummaryService(t, mockSrv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/segments/seg-1/summary", nil)
	srv.handleGetSummary(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleGetSummary_MissingID_Gap(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()
	srv := newTestServerWithSummaryService(t, mockSrv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/segments//summary", nil)
	srv.handleGetSummary(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleListSummaries_MethodNotAllowed_Gap(t *testing.T) {
	mockSrv := newMockAIServerForHandlerTest(t, "canned", 0)
	defer mockSrv.Close()
	srv := newTestServerWithSummaryService(t, mockSrv.URL)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

// --- removeChannel coverage ---

func TestRemoveChannel_NotFound(t *testing.T) {
	ch1 := make(chan BusEvent, 1)
	ch2 := make(chan BusEvent, 1)
	channels := []chan BusEvent{ch1}

	result := removeChannel(channels, ch2)
	if len(result) != 1 {
		t.Errorf("expected 1 channel, got %d", len(result))
	}
}

// --- handleIndex coverage ---

func TestHandleIndex_Gap(t *testing.T) {
	srv := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleIndex(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html content type, got %q", ct)
	}
}

// --- handleVersion coverage ---

func TestHandleVersion_Handler_Gap(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	srv.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool       `json:"success"`
		Data    VersionInfo `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

// --- InitMessage coverage ---

func TestInitMessage_Fields(t *testing.T) {
	msg := InitMessage{
		AuthToken: "test-token",
		Arguments: "?session=test",
	}
	if msg.AuthToken != "test-token" {
		t.Errorf("expected 'test-token', got %q", msg.AuthToken)
	}
	if msg.Arguments != "?session=test" {
		t.Errorf("expected '?session=test', got %q", msg.Arguments)
	}
}
