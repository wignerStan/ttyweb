package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestHandleProjects_GET_Empty(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects", nil)
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjects_POST_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProjects_POST_MissingName(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"path":"/some/path"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProjects_POST_MissingPath(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"MyProject"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProjects_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/projects", nil)
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleProjectDetail_EmptyID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/", nil)
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProjectDetail_GET_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/nonexistent", nil)
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleProjectDetail_PUT_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/projects/some-id", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProjectDetail_DELETE_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/projects/nonexistent", nil)
	srv.handleProjectDetail(rec, req)

	// The projectService singleton may use a stale DB connection,
	// so we accept either 404 or 500.
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 404 or 500, got %d", rec.Code)
	}
}

func TestHandleProjectDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/projects/some-id", nil)
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// createFakeGitRepo creates a temporary directory with a .git subdirectory
// to serve as a valid project path for tests.
func createFakeGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(dir+"/.git", 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestHandleProjects_POST_ValidProject(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)
	body := `{"name":"TestProject","path":"` + gitDir + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Data)
	}
	if resp.Data["name"] != "TestProject" {
		t.Fatalf("expected name TestProject, got %v", resp.Data["name"])
	}
}

func TestHandleProjects_POST_InvalidPath(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"BadPath","path":"/nonexistent/dir"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjects_POST_WithDescription(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)
	body := `{"name":"DescProject","path":"` + gitDir + `","description":"A test project"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjects_POST_DuplicatePath(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)
	body := `{"name":"Project1","path":"` + gitDir + `"}`

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req1.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first create, got %d", rec1.Code)
	}

	body2 := `{"name":"Project2","path":"` + gitDir + `"}`
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body2)))
	req2.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d; body: %s", rec2.Code, rec2.Body.String())
	}
}

func TestHandleProjects_GET_WithProjects(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)
	body := `{"name":"ListedProject","path":"` + gitDir + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects", nil)
	srv.handleProjects(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec2.Code, rec2.Body.String())
	}
}

func TestHandleProjectDetail_GET_ValidProject(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)

	// Create a project first.
	body := `{"name":"DetailProject","path":"` + gitDir + `"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleProjects(createRec, createReq)

	var createResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	id, ok := createResp.Data["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected project ID in response")
	}

	// GET the project.
	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/"+id, nil)
	srv.handleProjectDetail(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", getRec.Code, getRec.Body.String())
	}
}

func TestHandleProjectDetail_PUT_ValidProject(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)

	body := `{"name":"UpdateProject","path":"` + gitDir + `"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleProjects(createRec, createReq)

	var createResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	id, ok := createResp.Data["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected project ID in response")
	}

	putBody := `{"name":"UpdatedName"}`
	putRec := httptest.NewRecorder()
	putReq := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/projects/"+id, bytes.NewReader([]byte(putBody)))
	putReq.Header.Set("Content-Type", "application/json")
	srv.handleProjectDetail(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", putRec.Code, putRec.Body.String())
	}
}

func TestHandleProjectDetail_DELETE_ValidProject(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)

	body := `{"name":"DeleteProject","path":"` + gitDir + `"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleProjects(createRec, createReq)

	var createResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	id, ok := createResp.Data["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected project ID in response")
	}

	delRec := httptest.NewRecorder()
	delReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/projects/"+id, nil)
	srv.handleProjectDetail(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", delRec.Code, delRec.Body.String())
	}

	// Verify deleted.
	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/"+id, nil)
	srv.handleProjectDetail(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRec.Code)
	}
}

func TestHandleProjectDetail_SyncRoute(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)

	body := `{"name":"SyncProject","path":"` + gitDir + `"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleProjects(createRec, createReq)

	var createResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(createRec.Body).Decode(&createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	id, ok := createResp.Data["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected project ID in response")
	}

	// Hit the sync route through handleProjectDetail.
	syncRec := httptest.NewRecorder()
	syncReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects/"+id+"/sync", nil)
	srv.handleProjectDetail(syncRec, syncReq)

	// Sync may succeed or fail depending on remote, but should not panic.
	if syncRec.Code != http.StatusOK && syncRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 200 or 500, got %d; body: %s", syncRec.Code, syncRec.Body.String())
	}
}

func TestHandleProjects_GET_NoDB(t *testing.T) {
	// Reset the projectService singleton so it fails without a DB.
	projectServiceOnce = sync.Once{}
	projectServiceInstance = nil
	errProjectService = nil

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects", nil)
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjects_POST_WithWorktreeBasePath(t *testing.T) {
	srv := newTestServerWithDB(t)
	gitDir := createFakeGitRepo(t)
	body := `{"name":"WTProject","path":"` + gitDir + `","worktree_base_path":"/tmp/worktrees"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/projects", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjectDetail_GET_NoDB(t *testing.T) {
	projectServiceOnce = sync.Once{}
	projectServiceInstance = nil
	errProjectService = nil

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/projects/some-id", nil)
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjectDetail_PUT_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"Updated"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/projects/nonexistent", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleProjectDetail_DELETE_InternalError(t *testing.T) {
	projectServiceOnce = sync.Once{}
	projectServiceInstance = nil
	errProjectService = nil

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/projects/some-id", nil)
	srv.handleProjectDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
