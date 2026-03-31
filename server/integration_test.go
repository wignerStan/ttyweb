//go:build integration

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"ttyweb/service"
)

// Integration tests exercise full request/response cycles through the server
// handlers, backed by real SQLite. Each test verifies a complete CRUD lifecycle
// or cross-endpoint flow using unique data to avoid interference.

func postJSON(t *testing.T, handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handler(rec, req)
	return rec
}

func getJSON(t *testing.T, handler http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	handler(rec, req)
	return rec
}

func putJSON(t *testing.T, handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handler(rec, req)
	return rec
}

func deleteReq(t *testing.T, handler http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, path, nil)
	handler(rec, req)
	return rec
}

func patchJSON(t *testing.T, handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handler(rec, req)
	return rec
}

func decodeResp(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return m
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected %d, got %d; body: %s", want, rec.Code, rec.Body.String())
	}
}

// initTempGitRepo creates a temporary directory, initializes a git repo
// with an initial commit, and returns the absolute path.
func initTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	// Create an initial file and commit so branches can be created.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return dir
}

func TestIntegration_ProfileLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Create
	rec := postJSON(t, srv.handleProfiles, "/api/profiles",
		`{"profile_key":"integ-prof","name":"Original","sort_order":1}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// Verify in list
	rec = getJSON(t, srv.handleProfiles, "/api/profiles")
	assertStatus(t, rec, http.StatusOK)
	data := decodeResp(t, rec)["data"].([]any)
	found := false
	for _, item := range data {
		if int(item.(map[string]any)["id"].(float64)) == id {
			found = true
		}
	}
	if !found {
		t.Fatal("profile not in list")
	}

	// Update
	rec = putJSON(t, srv.handleProfileDetail, fmt.Sprintf("/api/profiles/%d", id),
		`{"profile_key":"integ-prof","name":"Updated","sort_order":5}`)
	assertStatus(t, rec, http.StatusOK)
	resp := decodeResp(t, rec)
	if resp["data"].(map[string]any)["name"] != "Updated" {
		t.Fatal("name not updated")
	}

	// Delete and verify gone
	rec = deleteReq(t, srv.handleProfileDetail, fmt.Sprintf("/api/profiles/%d", id))
	assertStatus(t, rec, http.StatusOK)
	for _, p := range store.ListProfiles() {
		if p.ID == id {
			t.Fatal("profile still exists")
		}
	}
}

func TestIntegration_GroupLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleGroups, "/api/groups",
		`{"group_name":"integ-grp","sort_order":1,"profile_key":"pk-integ"}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// Filter by profile_key
	rec = getJSON(t, srv.handleGroups, "/api/groups?profile_key=pk-integ")
	assertStatus(t, rec, http.StatusOK)
	items := decodeResp(t, rec)["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 group, got %d", len(items))
	}

	rec = putJSON(t, srv.handleGroupDetail, fmt.Sprintf("/api/groups/%d", id),
		`{"group_name":"renamed","sort_order":5,"profile_key":"pk-integ"}`)
	assertStatus(t, rec, http.StatusOK)

	rec = deleteReq(t, srv.handleGroupDetail, fmt.Sprintf("/api/groups/%d", id))
	assertStatus(t, rec, http.StatusOK)
	for _, g := range store.ListGroups("") {
		if g.ID == id {
			t.Fatal("group still exists")
		}
	}
}

func TestIntegration_SnippetLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleSnippets, "/api/snippets",
		`{"name":"integ-snip","command":"echo hi"}`)
	assertStatus(t, rec, http.StatusOK)
	idx := int(decodeResp(t, rec)["data"].(map[string]any)["index"].(float64))

	// Verify in list
	rec = getJSON(t, srv.handleSnippets, "/api/snippets")
	assertStatus(t, rec, http.StatusOK)

	rec = putJSON(t, srv.handleSnippetDetail, fmt.Sprintf("/api/snippets/%d", idx),
		`{"name":"integ-snip-v2","command":"echo bye"}`)
	assertStatus(t, rec, http.StatusOK)

	rec = deleteReq(t, srv.handleSnippetDetail, fmt.Sprintf("/api/snippets/%d", idx))
	assertStatus(t, rec, http.StatusOK)
	for _, s := range store.ListSnippets() {
		if s.Name == "integ-snip-v2" {
			t.Fatal("snippet still exists")
		}
	}
}

func TestIntegration_RoleLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleRoles, "/api/roles",
		`{"name":"IntegRole","description":"test","system_prompt":"Be precise"}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// List should have 7 builtin + 1 custom
	rec = getJSON(t, srv.handleRoles, "/api/roles")
	assertStatus(t, rec, http.StatusOK)
	if len(decodeResp(t, rec)["data"].([]any)) != 8 {
		t.Fatal("expected 8 roles total")
	}

	rec = putJSON(t, srv.handleRoleDetail, fmt.Sprintf("/api/roles/%d", id),
		`{"name":"EditedRole","description":"edited","system_prompt":"Think carefully"}`)
	assertStatus(t, rec, http.StatusOK)

	rec = deleteReq(t, srv.handleRoleDetail, fmt.Sprintf("/api/roles/%d", id))
	assertStatus(t, rec, http.StatusOK)
	for _, r := range store.ListRoles() {
		if r.ID == id {
			t.Fatal("role still exists")
		}
	}
}

func TestIntegration_TaskEventFlow(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleTaskDetail, "/api/tasks/events",
		`{"task_id":"itask","pane_key":"ipane","event":"user_message","data":{"text":"hi"}}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// Verify total in list
	rec = getJSON(t, srv.handleTasks, "/api/tasks")
	assertStatus(t, rec, http.StatusOK)
	if decodeResp(t, rec)["data"].(map[string]any)["total"] != float64(1) {
		t.Fatal("expected total 1")
	}

	// Complete
	rec = postJSON(t, srv.handleTaskDetail, fmt.Sprintf("/api/tasks/%d/complete", id), "")
	assertStatus(t, rec, http.StatusOK)
	events := store.GetTaskEventsByPane("ipane")
	if len(events) == 0 || !events[0].Completed {
		t.Fatal("task not completed")
	}
}

func TestIntegration_SessionEndpoints(t *testing.T) {
	srv := newTestServerWithDB(t)
	srv.factory = &mockFactory{name: "local command"}

	// GET /api/sessions — empty for local backend
	rec := getJSON(t, srv.handleListSessions, "/api/sessions")
	assertStatus(t, rec, http.StatusOK)

	// GET /api/backends — returns availability
	rec = getJSON(t, srv.handleListBackends, "/api/backends")
	assertStatus(t, rec, http.StatusOK)
	if len(decodeResp(t, rec)["data"].([]any)) != 3 {
		t.Fatal("expected 3 backends")
	}
}

func TestIntegration_AuthEndpoints(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := getJSON(t, srv.handleAuthCheck, "/api/auth/check")
	assertStatus(t, rec, http.StatusOK)
	resp := decodeResp(t, rec)
	if !resp["success"].(bool) {
		t.Fatal("expected success")
	}
	data := resp["data"].(map[string]any)
	if !data["authenticated"].(bool) {
		t.Fatal("expected authenticated=true")
	}
}

func TestIntegration_PaneStatusFlow(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Post event with pane_key
	postJSON(t, srv.handleTaskDetail, "/api/tasks/events",
		`{"task_id":"ipane-task","pane_key":"ipane-status","event":"msg","data":{"text":"ok"}}`)

	// Set pane status
	rec := putJSON(t, srv.handlePaneStatus, "/api/panes/status",
		`{"pane_key":"ipane-status","status":"running"}`)
	assertStatus(t, rec, http.StatusOK)

	// Verify pane in status list
	rec = getJSON(t, srv.handlePaneStatus, "/api/panes/status")
	assertStatus(t, rec, http.StatusOK)
	data := decodeResp(t, rec)["data"].(map[string]any)
	if data["ipane-status"] != "running" {
		t.Fatalf("expected pane status 'running', got %v", data["ipane-status"])
	}
}

// ---------------------------------------------------------------------------
// Version endpoint
// ---------------------------------------------------------------------------

func TestIntegration_Version(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := getJSON(t, srv.handleVersion, "/api/version")
	assertStatus(t, rec, http.StatusOK)
	resp := decodeResp(t, rec)
	if !resp["success"].(bool) {
		t.Fatal("expected success")
	}
	d := resp["data"].(map[string]any)
	if d["version"] == nil || d["version"] == "" {
		t.Fatal("expected non-empty version")
	}
	if d["goVersion"] == nil || d["goVersion"] == "" {
		t.Fatal("expected non-empty goVersion")
	}
}

// ---------------------------------------------------------------------------
// Branch API
// ---------------------------------------------------------------------------

func TestIntegration_BranchLifecycle(t *testing.T) {
	repoPath := initTempGitRepo(t)

	// List branches — should have "main" at minimum.
	rec := getJSON(t, handleBranches, "/api/branches?repo="+repoPath)
	assertStatus(t, rec, http.StatusOK)
	branches := decodeResp(t, rec)["data"].([]any)
	if len(branches) < 1 {
		t.Fatal("expected at least 1 branch (main)")
	}

	// Create a new branch.
	rec = postJSON(t, handleBranches, "/api/branches?repo="+repoPath,
		`{"name":"integ-branch"}`)
	assertStatus(t, rec, http.StatusOK)
	resp := decodeResp(t, rec)
	if resp["data"].(map[string]any)["name"] != "integ-branch" {
		t.Fatal("branch name mismatch")
	}

	// Verify it appears in the list.
	rec = getJSON(t, handleBranches, "/api/branches?repo="+repoPath)
	assertStatus(t, rec, http.StatusOK)
	branches = decodeResp(t, rec)["data"].([]any)
	found := false
	for _, b := range branches {
		if b.(map[string]any)["name"] == "integ-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created branch not in list")
	}

	// Delete the branch.
	rec = deleteReq(t, handleBranchDetail, "/api/branches/integ-branch?repo="+repoPath)
	assertStatus(t, rec, http.StatusOK)

	// Verify it is gone.
	rec = getJSON(t, handleBranches, "/api/branches?repo="+repoPath)
	assertStatus(t, rec, http.StatusOK)
	branches = decodeResp(t, rec)["data"].([]any)
	for _, b := range branches {
		if b.(map[string]any)["name"] == "integ-branch" {
			t.Fatal("branch still exists after delete")
		}
	}
}

func TestIntegration_BranchValidation(t *testing.T) {
	repoPath := initTempGitRepo(t)

	// Missing repo parameter.
	rec := getJSON(t, handleBranches, "/api/branches")
	assertStatus(t, rec, http.StatusBadRequest)

	// Empty branch name.
	rec = postJSON(t, handleBranches, "/api/branches?repo="+repoPath,
		`{"name":""}`)
	assertStatus(t, rec, http.StatusBadRequest)

	// Invalid branch name with spaces.
	rec = postJSON(t, handleBranches, "/api/branches?repo="+repoPath,
		`{"name":"invalid name"}`)
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// File Browser API
// ---------------------------------------------------------------------------

func TestIntegration_FileBrowserList(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Register a temp directory as an allowed FS root.
	tmpDir := t.TempDir()
	// Create a known file and subdirectory.
	if err := os.WriteFile(filepath.Join(tmpDir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	registerFSRoot(tmpDir)

	// List the registered directory.
	rec := getJSON(t, srv.handleFSList, "/api/fs?path="+tmpDir)
	assertStatus(t, rec, http.StatusOK)
	entries := decodeResp(t, rec)["data"].([]any)
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(entries))
	}

	// Verify "hello.txt" and "subdir" are present.
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.(map[string]any)["name"].(string)] = true
	}
	if !names["hello.txt"] {
		t.Fatal("missing hello.txt")
	}
	if !names["subdir"] {
		t.Fatal("missing subdir")
	}
}

func TestIntegration_FileBrowserForbidden(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Try to list a path that is not registered.
	rec := getJSON(t, srv.handleFSList, "/api/fs?path=/nonexistent-allowed-path")
	assertStatus(t, rec, http.StatusForbidden)
}

// ---------------------------------------------------------------------------
// Stats API
// ---------------------------------------------------------------------------

func TestIntegration_StatsServiceUnavailable(t *testing.T) {
	srv := newTestServerWithDB(t)
	// statsService is nil by default in newTestServerWithDB.

	rec := getJSON(t, srv.handleTaskStats, "/api/tasks/stats")
	assertStatus(t, rec, http.StatusServiceUnavailable)
	resp := decodeResp(t, rec)
	if resp["success"].(bool) {
		t.Fatal("expected failure")
	}
}

func TestIntegration_StatsWithDaysParam(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Even with a days parameter, nil service should return 503.
	rec := getJSON(t, srv.handleTaskStats, "/api/tasks/stats?days=30")
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ---------------------------------------------------------------------------
// Summary API
// ---------------------------------------------------------------------------

func TestIntegration_SummarizeServiceUnavailable(t *testing.T) {
	srv := newTestServerWithDB(t)
	// summaryService is nil by default in newTestServerWithDB.

	// POST /api/segments/1/summarize
	rec := postJSON(t, srv.handleSummarizeSegment, "/api/segments/1/summarize", "")
	assertStatus(t, rec, http.StatusServiceUnavailable)

	// GET /api/segments/1/summary
	rec = getJSON(t, srv.handleGetSummary, "/api/segments/1/summary")
	assertStatus(t, rec, http.StatusServiceUnavailable)

	// GET /api/segments/summaries
	rec = getJSON(t, srv.handleListSummaries, "/api/segments/summaries")
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// ---------------------------------------------------------------------------
// Task AI Session linking
// ---------------------------------------------------------------------------

func TestIntegration_TaskAISessionLinkFlow(t *testing.T) {
	srv := newTestServerWithDB(t)
	// Reset the global task-AI session service to avoid cross-test state.
	taskAISessionSvc = service.NewTaskAISessionService()

	taskID := "integ-task-ai"
	aiSessionID := "session-42"

	// List linked sessions — should be empty.
	rec := getJSON(t, srv.handleTaskAISessionLinks,
		fmt.Sprintf("/api/kanban/tasks/%s/sessions", taskID))
	assertStatus(t, rec, http.StatusOK)
	links := decodeResp(t, rec)["data"].([]any)
	if len(links) != 0 {
		t.Fatalf("expected 0 linked sessions, got %d", len(links))
	}

	// Link an AI session.
	rec = postJSON(t, srv.handleTaskAISessionLinks,
		fmt.Sprintf("/api/kanban/tasks/%s/sessions", taskID),
		fmt.Sprintf(`{"ai_session_id":"%s"}`, aiSessionID))
	assertStatus(t, rec, http.StatusOK)

	// Verify the link appears in the list.
	rec = getJSON(t, srv.handleTaskAISessionLinks,
		fmt.Sprintf("/api/kanban/tasks/%s/sessions", taskID))
	assertStatus(t, rec, http.StatusOK)
	links = decodeResp(t, rec)["data"].([]any)
	if len(links) != 1 {
		t.Fatalf("expected 1 linked session, got %d", len(links))
	}

	// Unlink the session.
	rec = deleteReq(t, srv.handleTaskAISessionLinks,
		fmt.Sprintf("/api/kanban/tasks/%s/sessions/%s", taskID, aiSessionID))
	assertStatus(t, rec, http.StatusOK)

	// Verify it is gone.
	rec = getJSON(t, srv.handleTaskAISessionLinks,
		fmt.Sprintf("/api/kanban/tasks/%s/sessions", taskID))
	assertStatus(t, rec, http.StatusOK)
	links = decodeResp(t, rec)["data"].([]any)
	if len(links) != 0 {
		t.Fatalf("expected 0 linked sessions after unlink, got %d", len(links))
	}
}

// ---------------------------------------------------------------------------
// Project CRUD
// ---------------------------------------------------------------------------

func TestIntegration_ProjectLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)
	repoPath := initTempGitRepo(t)

	// Create a project.
	rec := postJSON(t, srv.handleProjects, "/api/projects",
		fmt.Sprintf(`{"name":"IntegProject","path":"%s","description":"test project"}`, repoPath))
	assertStatus(t, rec, http.StatusCreated)
	project := decodeResp(t, rec)["data"].(map[string]any)
	projectID := project["id"].(string)

	if project["name"] != "IntegProject" {
		t.Fatalf("expected name IntegProject, got %v", project["name"])
	}

	// List projects — should contain the created project.
	rec = getJSON(t, srv.handleProjects, "/api/projects")
	assertStatus(t, rec, http.StatusOK)
	projects := decodeResp(t, rec)["data"].([]any)
	found := false
	for _, p := range projects {
		if p.(map[string]any)["id"] == projectID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("project not in list")
	}

	// Get project detail.
	rec = getJSON(t, srv.handleProjectDetail, fmt.Sprintf("/api/projects/%s", projectID))
	assertStatus(t, rec, http.StatusOK)
	detail := decodeResp(t, rec)["data"].(map[string]any)
	if detail["name"] != "IntegProject" {
		t.Fatal("project detail name mismatch")
	}

	// Update project.
	rec = putJSON(t, srv.handleProjectDetail, fmt.Sprintf("/api/projects/%s", projectID),
		`{"name":"UpdatedProject"}`)
	assertStatus(t, rec, http.StatusOK)
	updated := decodeResp(t, rec)["data"].(map[string]any)
	if updated["name"] != "UpdatedProject" {
		t.Fatal("project not updated")
	}

	// Delete project.
	rec = deleteReq(t, srv.handleProjectDetail, fmt.Sprintf("/api/projects/%s", projectID))
	assertStatus(t, rec, http.StatusOK)

	// Verify it is gone.
	rec = getJSON(t, srv.handleProjectDetail, fmt.Sprintf("/api/projects/%s", projectID))
	assertStatus(t, rec, http.StatusNotFound)
}

func TestIntegration_ProjectValidation(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Missing name.
	rec := postJSON(t, srv.handleProjects, "/api/projects",
		`{"name":"","path":"/tmp"}`)
	assertStatus(t, rec, http.StatusBadRequest)

	// Missing path.
	rec = postJSON(t, srv.handleProjects, "/api/projects",
		`{"name":"NoPath","path":""}`)
	assertStatus(t, rec, http.StatusBadRequest)

	// Invalid path (not a git repo).
	rec = postJSON(t, srv.handleProjects, "/api/projects",
		`{"name":"BadPath","path":"/tmp"}`)
	assertStatus(t, rec, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// Notepad CRUD
// ---------------------------------------------------------------------------

func TestIntegration_NotepadLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Create a note.
	rec := postJSON(t, srv.handleNotepad, "/api/notepad",
		`{"name":"integ-note","content":"Hello, integration test!"}`)
	assertStatus(t, rec, http.StatusOK)
	note := decodeResp(t, rec)["data"].(map[string]any)
	noteID := note["id"].(string)

	if note["name"] != "integ-note" {
		t.Fatalf("expected name integ-note, got %v", note["name"])
	}

	// List notes — should contain the created note.
	rec = getJSON(t, srv.handleNotepad, "/api/notepad")
	assertStatus(t, rec, http.StatusOK)
	notes := decodeResp(t, rec)["data"].([]any)
	found := false
	for _, n := range notes {
		if n.(map[string]any)["id"] == noteID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("note not in list")
	}

	// Get note detail.
	rec = getJSON(t, srv.handleNotepadDetail, fmt.Sprintf("/api/notepad/%s", noteID))
	assertStatus(t, rec, http.StatusOK)
	detail := decodeResp(t, rec)["data"].(map[string]any)
	if detail["name"] != "integ-note" {
		t.Fatal("note detail name mismatch")
	}

	// Update note.
	rec = putJSON(t, srv.handleNotepadDetail, fmt.Sprintf("/api/notepad/%s", noteID),
		`{"name":"updated-note","content":"Updated content"}`)
	assertStatus(t, rec, http.StatusOK)
	updated := decodeResp(t, rec)["data"].(map[string]any)
	if updated["name"] != "updated-note" {
		t.Fatal("note name not updated")
	}
	if updated["content"] != "Updated content" {
		t.Fatal("note content not updated")
	}

	// Delete note.
	rec = deleteReq(t, srv.handleNotepadDetail, fmt.Sprintf("/api/notepad/%s", noteID))
	assertStatus(t, rec, http.StatusOK)

	// Verify it is gone.
	rec = getJSON(t, srv.handleNotepadDetail, fmt.Sprintf("/api/notepad/%s", noteID))
	assertStatus(t, rec, http.StatusNotFound)
}

func TestIntegration_NotepadFilterByProject(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Create two notes: one with a project_id, one without.
	rec := postJSON(t, srv.handleNotepad, "/api/notepad",
		`{"name":"with-project","content":"proj","project_id":"proj-1"}`)
	assertStatus(t, rec, http.StatusOK)

	postJSON(t, srv.handleNotepad, "/api/notepad",
		`{"name":"no-project","content":"none"}`)

	// Filter by project_id.
	rec = getJSON(t, srv.handleNotepad, "/api/notepad?project_id=proj-1")
	assertStatus(t, rec, http.StatusOK)
	notes := decodeResp(t, rec)["data"].([]any)
	if len(notes) != 1 {
		t.Fatalf("expected 1 note with project_id, got %d", len(notes))
	}
}

func TestIntegration_NotepadCreateInvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Send invalid JSON.
	rec := postJSON(t, srv.handleNotepad, "/api/notepad", `{invalid}`)
	assertStatus(t, rec, http.StatusBadRequest)
}
