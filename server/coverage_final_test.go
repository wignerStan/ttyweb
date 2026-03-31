package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"ttyweb/db"
	"ttyweb/service"
)

// --- version.go ---

func TestHandleVersion_Success(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/version", nil)
	srv.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestGetAppVersion_Fields(t *testing.T) {
	info := getAppVersion()
	if info.Version == "" {
		t.Error("expected non-empty Version")
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
}

// --- ws_ai_stream.go ---

func TestResolveSystemPrompt_KnownRoleWithStore(t *testing.T) {
	store = NewMemoryStore()
	// The "cli" role maps to builtin ID 1, which already exists in store.
	// Verify the default system prompt is returned when the role exists in store.
	prompt := resolveSystemPrompt("cli")
	if prompt == "" {
		t.Error("expected non-empty prompt for known role 'cli'")
	}
	// The exact prompt depends on the builtin roles loaded at init.
}

func TestEnvOrDefault_Set(t *testing.T) {
	t.Setenv("TEST_COV_VAR", "hello")
	got := envOrDefault("TEST_COV_VAR", "default")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestEnvOrDefault_Unset(t *testing.T) {
	got := envOrDefault("NONEXISTENT_COV_VAR_12345", "default")
	if got != "default" {
		t.Errorf("expected 'default', got %q", got)
	}
}

func TestSanitizeStreamError(t *testing.T) {
	msg := sanitizeStreamError(nil)
	if msg != "AI stream error" {
		t.Errorf("expected 'AI stream error', got %q", msg)
	}
}

// --- handlers.go ---

func TestWebttyOptions_AllEnabled(t *testing.T) {
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
	if len(opts) != 5 {
		t.Errorf("expected 5 options, got %d", len(opts))
	}
}

func TestWebttyOptions_Minimal(t *testing.T) {
	srv := &Server{
		options: &Options{},
	}
	opts := srv.webttyOptions([]byte("title"))
	if len(opts) != 1 {
		t.Errorf("expected 1 option (title), got %d", len(opts))
	}
}

func TestParseInitArguments_InvalidSession(t *testing.T) {
	srv := &Server{
		options: &Options{PermitArguments: true},
	}
	init := &InitMessage{Arguments: "?session=bad name!"}
	_, err := srv.parseInitArguments(init)
	if err == nil {
		t.Error("expected error for invalid session name")
	}
}

func TestParseInitArguments_InvalidPane(t *testing.T) {
	srv := &Server{
		options: &Options{PermitArguments: true},
	}
	init := &InitMessage{Arguments: "?pane=bad pane!"}
	_, err := srv.parseInitArguments(init)
	if err == nil {
		t.Error("expected error for invalid pane ID")
	}
}

func TestParseInitArguments_NoPermitArguments(t *testing.T) {
	srv := &Server{
		options: &Options{PermitArguments: false},
	}
	init := &InitMessage{Arguments: "?session=test&pane=0"}
	params, err := srv.parseInitArguments(init)
	if err != nil {
		t.Fatalf("parseInitArguments: %v", err)
	}
	if params.Get("session") != "" {
		t.Error("expected no session when PermitArguments is false")
	}
}

// --- api_project.go ---

func TestDeleteProject_Success_Final(t *testing.T) {
	srv := newTestServerWithDB(t)
	gormDB, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewProjectService(gormDB)
	srv.options.Path = "/"

	tmpDir := t.TempDir()
	mustGitInit(t, tmpDir)
	project, err := svc.AddProject("test-proj-final", tmpDir)
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	rec := httptest.NewRecorder()
	deleteProject(rec, svc, project.ID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// --- api_ai_session.go ---

func TestHandleAICleanupSessions_MethodNotAllowed_Final(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/cleanup", nil)
	srv.handleAICleanupSessions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAISessionSubroute_InvalidID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/not-a-number", nil)
	srv.handleAISessionSubroute(rec, req, "not-a-number")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAISessionSubroute_UnknownRoute(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/1/unknown", nil)
	srv.handleAISessionSubroute(rec, req, "1/unknown")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAISessionDetail_MethodNotAllowed_Final(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/1", nil)
	srv.aiSessionDetail(rec, req, 1)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestAISessionDetail_NotFound_Final(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/99999", nil)
	srv.aiSessionDetail(rec, req, 99999)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAISessionConversationOrRefresh_MethodNotAllowed_Final(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/ai/sessions/1/conversation", nil)
	srv.aiSessionConversationOrRefresh(rec, req, 1, false)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestAISessionConversationOrRefresh_NotFound_Final(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ai/sessions/99999/conversation", nil)
	srv.aiSessionConversationOrRefresh(rec, req, 99999, false)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- api_stats.go ---

func TestHandleTaskStats_NoService_Final(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/stats", nil)
	srv.handleTaskStats(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

// --- update_checker.go ---

func TestCheckForUpdatesURL_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	result, err := checkForUpdatesURL(context.Background(), "1.0.0", ts.URL)
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}
	if result.HasUpdate {
		t.Error("expected no update for 404 response")
	}
	if result.CurrentVersion != "1.0.0" {
		t.Errorf("expected CurrentVersion '1.0.0', got %q", result.CurrentVersion)
	}
}

func TestCheckForUpdatesURL_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()

	_, err := checkForUpdatesURL(context.Background(), "1.0.0", ts.URL)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestCheckForUpdatesURL_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	_, err := checkForUpdatesURL(context.Background(), "1.0.0", ts.URL)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestCheckForUpdatesURL_HasUpdate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v2.0.0",
			"html_url": "https://github.com/owner/repo/releases/tag/v2.0.0",
		})
	}))
	defer ts.Close()

	result, err := checkForUpdatesURL(context.Background(), "1.0.0", ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasUpdate {
		t.Error("expected HasUpdate=true when latest > current")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("expected '2.0.0', got %q", result.LatestVersion)
	}
}

func TestCheckForUpdatesURL_NoUpdate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v1.0.0",
			"html_url": "https://github.com/owner/repo/releases/tag/v1.0.0",
		})
	}))
	defer ts.Close()

	result, err := checkForUpdatesURL(context.Background(), "1.0.0", ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasUpdate {
		t.Error("expected HasUpdate=false when versions match")
	}
}

// --- api_notepad.go ---

// (TestHandleNotepad_MethodNotAllowed exists in api_notepad_test.go)

// --- api_worktree.go ---

func TestHandleWorktreeProjects_Post_EmptyPath(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects",
		strings.NewReader(`{"path":""}`))
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeProjects_Post_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects",
		strings.NewReader("not json"))
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- helpers ---

func mustGitInit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", dir) //nolint:gosec,noctx // test subprocess
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
}

// Ensure time import is used.
var _ = time.Now
