package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"ttyweb/worktree"
)

func TestHandleWorktreeProjects_GET(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects", nil)
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleWorktreeProjects_POST_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeProjects_POST_MissingPath(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeProjects_POST_EmptyPath(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects", bytes.NewReader([]byte(`{"path":"  "}`)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeProjects_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects", nil)
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWorktreeList_GET(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1/worktrees", nil)
	handleWorktreeList(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (no project), got %d", rec.Code)
	}
}

func TestHandleWorktreeList_POST_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeList(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeList_POST_MissingBranchName(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeList(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeList_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects/p1/worktrees", nil)
	handleWorktreeList(rec, req, "p1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWorktreeSync_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1/worktrees/sync", nil)
	handleWorktreeSync(rec, req, "p1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestMakeProjectDetailHandler_EmptyProjectID(t *testing.T) {
	handler := makeProjectDetailHandler("/api/worktree/projects")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/", nil)
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMakeProjectDetailHandler_NotFound(t *testing.T) {
	handler := makeProjectDetailHandler("/api/worktree/projects")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1", nil)
	handler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleWorktreeItem_DELETE(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects/p1/worktrees/wt1", nil)
	handleWorktreeItem(rec, req, "p1", "wt1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (no project), got %d", rec.Code)
	}
}

func TestHandleWorktreeItem_POST_Refresh(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees/wt1/refresh", nil)
	handleWorktreeItem(rec, req, "p1", "wt1/refresh")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (no project), got %d", rec.Code)
	}
}

func TestHandleWorktreeItem_POST_Commit_MissingMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees/wt1/commit", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeItem(rec, req, "p1", "wt1/commit")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeItem_POST_Commit_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees/wt1/commit", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeItem(rec, req, "p1", "wt1/commit")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeItem_POST_UnknownAction(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees/wt1/unknown", nil)
	handleWorktreeItem(rec, req, "p1", "wt1/unknown")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeItem_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1/worktrees/wt1", nil)
	handleWorktreeItem(rec, req, "p1", "wt1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// createRealGitRepo creates a temporary directory and runs `git init` in it.
func createRealGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = dir
	cmd.Env = worktree.FilterGitEnv(os.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v; output: %s", err, out)
	}
	return dir
}

func TestHandleWorktreeProjects_POST_ValidGitRepo(t *testing.T) {
	gitDir := createRealGitRepo(t)
	body := `{"path":"` + gitDir + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}
}

func TestHandleWorktreeProjects_POST_NonexistentPath(t *testing.T) {
	body := `{"path":"/nonexistent/path/12345"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorktreeProjects_POST_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	body := `{"path":"` + dir + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMakeProjectDetailHandler_WorktreesRoute(t *testing.T) {
	handler := makeProjectDetailHandler("/api/worktree/projects")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1/worktrees", nil)
	handler(rec, req)

	// Project not found returns 400 from the service.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMakeProjectDetailHandler_WorktreesSyncRoute(t *testing.T) {
	handler := makeProjectDetailHandler("/api/worktree/projects")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees/sync", nil)
	handler(rec, req)

	// Project not found returns 400 from the service.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMakeProjectDetailHandler_WorktreeItemRoute(t *testing.T) {
	handler := makeProjectDetailHandler("/api/worktree/projects")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects/p1/worktrees/wt1", nil)
	handler(rec, req)

	// Project not found returns 400 from the service.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleWorktreeSync_WithRegisteredProject(t *testing.T) {
	gitDir := createRealGitRepo(t)
	project, err := wtService.AddProject(gitDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/"+project.ID+"/worktrees/sync", nil)
	handleWorktreeSync(rec, req, project.ID)

	// Sync should succeed (no worktrees to sync).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorktreeList_WithRegisteredProject(t *testing.T) {
	gitDir := createRealGitRepo(t)
	project, err := wtService.AddProject(gitDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/"+project.ID+"/worktrees", nil)
	handleWorktreeList(rec, req, project.ID)

	// Should return empty list (no worktrees).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorktreeItem_DELETE_RegisteredProject(t *testing.T) {
	gitDir := createRealGitRepo(t)
	project, err := wtService.AddProject(gitDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects/"+project.ID+"/worktrees/nonexistent", nil)
	handleWorktreeItem(rec, req, project.ID, "nonexistent")

	// Worktree not found returns error.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorktreeItem_POST_Commit_RegisteredProject(t *testing.T) {
	gitDir := createRealGitRepo(t)
	project, err := wtService.AddProject(gitDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	body := `{"message":"test commit"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/"+project.ID+"/worktrees/nonexistent/commit", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeItem(rec, req, project.ID, "nonexistent/commit")

	// Worktree not found returns error.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleWorktreeList_POST_CreateWorktree(t *testing.T) {
	gitDir := createRealGitRepo(t)

	// Create an initial commit so worktrees can be created.
	runGit := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = gitDir
		cmd.Env = worktree.FilterGitEnv(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v; output: %s", args, err, out)
		}
	}
	runGit("config", "user.email", "test@test.com")
	runGit("config", "user.name", "Test")
	runGit("commit", "--allow-empty", "-m", "initial")

	project, err := wtService.AddProject(gitDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	body := `{"branchName":"test-branch","createBranch":true}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/"+project.ID+"/worktrees", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handleWorktreeList(rec, req, project.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// Parse the response to get the worktree ID.
	var createResp struct {
		Success bool                   `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	wtID, ok := createResp.Data["id"].(string)
	if !ok || wtID == "" {
		t.Fatal("expected worktree id in response")
	}

	// Delete the worktree.
	delRec := httptest.NewRecorder()
	delReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects/"+project.ID+"/worktrees/"+wtID, nil)
	handleWorktreeItem(delRec, delReq, project.ID, wtID)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d; body: %s", delRec.Code, delRec.Body.String())
	}
}
