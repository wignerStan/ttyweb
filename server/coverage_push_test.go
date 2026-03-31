package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ttyweb/config"
	"ttyweb/db"
	"ttyweb/service"
	"ttyweb/webtty"
)

// --- classifyWSCloseError additional coverage ---

func TestClassifyWSCloseError_SpecialChars(t *testing.T) {
	got := classifyWSCloseError(errors.New("special: chars & more"), "tmux")
	if !strings.Contains(got, "an error: special: chars & more") {
		t.Errorf("expected error with special chars, got %q", got)
	}
}

// --- EventBus() getter ---

func TestEventBus_ReturnsInstance(t *testing.T) {
	bus := NewTaskEventBus()
	srv := &Server{eventBus: bus}

	got := srv.EventBus()
	if got != bus {
		t.Error("EventBus() should return the same bus instance")
	}
}

// --- handleAICommitMessage error paths ---

func TestHandleAICommitMessage_ProjectNotFound(t *testing.T) {
	store = NewMemoryStore()
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	wtService = service.NewWorktreeService()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/nonexistent/worktrees/w1/ai-commit-message", nil)
	handleAICommitMessage(rec, req, "nonexistent", "w1")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project, got %d", rec.Code)
	}
}

func TestFindWorktreeRecord_NotFound(t *testing.T) {
	store = NewMemoryStore()
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	wtService = service.NewWorktreeService()

	_, err := findWorktreeRecord(context.Background(), "nonexistent", "nonexistent-wt")
	if err == nil {
		t.Fatal("expected error for nonexistent worktree record")
	}
	// The error can be "project not found" (from ListWorktrees) or "worktree not found".
}

// --- handleAIStream additional error paths ---

func TestHandleAIStream_APIURLInvalid(t *testing.T) {
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	t.Setenv("OPENAI_API_URL", "not-a-valid-url")
	t.Setenv("OPENAI_API_KEY", "test-key")

	factory := &mockFactory{name: "test"}
	srv, err := New(factory, &Options{Path: "/", TitleFormat: "{{ .server.Version }}", Credential: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/ai/stream", srv.handleAIStream)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/ai/stream"
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	defer func() { _ = ws.Close() }()

	reqBody := map[string]string{
		"role":       "cli",
		"prompt":     "hello",
		"auth_token": "",
	}
	data, _ := json.Marshal(reqBody)
	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}

	var streamMsg wsStreamMessage
	if err := json.Unmarshal(msg, &streamMsg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if streamMsg.Type != "error" {
		t.Errorf("expected error type, got %q", streamMsg.Type)
	}
}

// --- cleanupLoop coverage ---

func TestCleanupLoop_ManualInvocation(t *testing.T) {
	t.Parallel()
	limiter := newVisitorLimiter(0, 5)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/health", nil)
	req.RemoteAddr = "10.99.99.1:1234"
	rec := httptest.NewRecorder()
	rateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	// Age out the visitor.
	limiter.mu.Lock()
	for _, v := range limiter.visitors {
		v.lastSeen = time.Now().Add(-4 * time.Minute).Unix()
	}
	limiter.mu.Unlock()

	// Manually invoke the cleanup logic (same as cleanupLoop body).
	limiter.mu.Lock()
	now := time.Now().Unix()
	for ip, v := range limiter.visitors {
		if now-v.lastSeen > int64(rateLimitCleanupInterval.Seconds()) {
			delete(limiter.visitors, ip)
		}
	}
	limiter.mu.Unlock()

	limiter.mu.Lock()
	count := len(limiter.visitors)
	limiter.mu.Unlock()
	if count != 0 {
		t.Errorf("expected 0 visitors after cleanup, got %d", count)
	}
}

// --- handleGetSummary / handleListSummaries / handleSummarizeSegment ---

func TestHandleGetSummary_NilService(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/123/summary", nil)
	srv.handleGetSummary(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for nil summary service, got %d", rec.Code)
	}
}

func TestHandleGetSummary_MethodNotAllowed_WithService(t *testing.T) {
	gormDB, _ := db.GetDB()
	svc := service.NewSummaryService(gormDB, nil)
	srv := &Server{
		options:         &Options{Path: "/"},
		summaryService: svc,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/123/summary", nil)
	srv.handleGetSummary(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleGetSummary_MissingSegmentID(t *testing.T) {
	gormDB, _ := db.GetDB()
	svc := service.NewSummaryService(gormDB, nil)
	srv := &Server{
		options:         &Options{Path: "/"},
		summaryService: svc,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments//summary", nil)
	srv.handleGetSummary(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing segment ID, got %d", rec.Code)
	}
}

func TestHandleListSummaries_NilService(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for nil summary service, got %d", rec.Code)
	}
}

func TestHandleListSummaries_MethodNotAllowed_WithService(t *testing.T) {
	gormDB, _ := db.GetDB()
	svc := service.NewSummaryService(gormDB, nil)
	srv := &Server{
		options:         &Options{Path: "/"},
		summaryService: svc,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/summaries", nil)
	srv.handleListSummaries(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSummarizeSegment_NilService(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/123/summarize", nil)
	srv.handleSummarizeSegment(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for nil summary service, got %d", rec.Code)
	}
}

// --- deleteProject error paths ---

func TestDeleteProject_NotFound(t *testing.T) {
	store = NewMemoryStore()
	path := t.TempDir() + "/test.db"
	if err := db.Init(path); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewProjectService(gormDB)

	rec := httptest.NewRecorder()
	deleteProject(rec, svc, "nonexistent-id")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent project, got %d", rec.Code)
	}
}

// --- resolveSystemPrompt with all known roles ---

func TestResolveSystemPrompt_AllRoles(t *testing.T) {
	store = NewMemoryStore()
	for roleID, storeID := range builtinRoleIDMap {
		prompt := resolveSystemPrompt(roleID)
		if prompt == "" {
			t.Errorf("expected non-empty prompt for role %q (store ID %d)", roleID, storeID)
		}
	}
}

// --- Config helper: NewCommitMessageService ---

func TestNewCommitMessageService_NilConfig(t *testing.T) {
	svc := service.NewCommitMessageService(nil)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewCommitMessageService_EmptyAPIKey(t *testing.T) {
	cfg := &config.Config{}
	svc := service.NewCommitMessageService(cfg)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

// --- envOrDefault with empty value ---

func TestEnvOrDefault_EmptyValue(t *testing.T) {
	t.Setenv("TEST_EMPTY_VAR_COVERAGE", "")
	got := envOrDefault("TEST_EMPTY_VAR_COVERAGE", "default")
	if got != "default" {
		t.Errorf("expected default for empty env var, got %q", got)
	}
}

// --- webtty.ErrSlaveClosed and ErrMasterClosed classification ---

func TestClassifyWSCloseError_BackendNames(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		backendName string
		want        string
	}{
		{"slave closed zellij", webtty.ErrSlaveClosed, "zellij", "zellij"},
		{"master closed local", webtty.ErrMasterClosed, "local", "client"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyWSCloseError(tc.err, tc.backendName)
			if got != tc.want {
				t.Errorf("classifyWSCloseError() = %q, want %q", got, tc.want)
			}
		})
	}
}
