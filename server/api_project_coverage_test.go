package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGolden_DeleteProject_NotFound covers the not-found error path in deleteProject.
func TestGolden_DeleteProject_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/projects/nonexistent-id", nil,
	)
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_DeleteProject_Success covers the successful deletion path.
func TestGolden_DeleteProject_Success(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Create a real git repo for the project path.
	projectDir := t.TempDir()
	mustGit(t, projectDir, "init", "-b", "main")
	mustGit(t, projectDir, "config", "user.name", "test")
	mustGit(t, projectDir, "config", "user.email", "test@test.com")

	// First create a project using the POST endpoint.
	createBody := bytes.NewReader([]byte(`{"name":"test-proj-del","path":"` + projectDir + `"}`))
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		createBody,
	)
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleProjects(createRec, createReq)

	// Extract the project ID from the response.
	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	projID := createResp.Data.ID
	if projID == "" {
		t.Fatalf("expected non-empty project ID, got response: %s", createRec.Body.String())
	}

	// Now delete it.
	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/projects/"+projID, nil,
	)
	srv.handleProjectDetail(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	// Verify response contains "deleted" status.
	body := deleteRec.Body.String()
	if body == "" || body[0] != '{' {
		t.Fatalf("expected JSON response, got: %s", body)
	}
}

// TestGolden_ProjectDetail_NoID covers missing project ID path.
func TestGolden_ProjectDetail_NoID(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/projects/", nil,
	)
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectSync_MethodNotAllowed covers non-POST on sync endpoint.
func TestGolden_ProjectSync_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/projects/some-id/sync", nil,
	)
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectSync_NotFound covers not-found on sync endpoint.
func TestGolden_ProjectSync_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects/nonexistent-id/sync", nil,
	)
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectMethodNotAllowed covers unsupported methods.
func TestGolden_ProjectMethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPatch,
		"/api/projects/some-id", nil,
	)
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectListMethodNotAllowed covers unsupported methods on list endpoint.
func TestGolden_ProjectListMethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodDelete,
		"/api/projects", nil,
	)
	srv.handleProjects(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectCreate_BadJSON covers invalid JSON body.
func TestGolden_ProjectCreate_BadJSON(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		bytes.NewReader([]byte("not json")),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectUpdate_NotFound covers update for nonexistent project.
func TestGolden_ProjectUpdate_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPut,
		"/api/projects/nonexistent-id",
		bytes.NewReader([]byte(`{"name":"updated"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectUpdate_BadJSON covers invalid JSON on update.
func TestGolden_ProjectUpdate_BadJSON(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPut,
		"/api/projects/some-id",
		bytes.NewReader([]byte("not json")),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjectDetail(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectCreate_MissingName covers missing name error.
func TestGolden_ProjectCreate_MissingName(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		bytes.NewReader([]byte(`{"path":"/some/path"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectCreate_MissingPath covers missing path error.
func TestGolden_ProjectCreate_MissingPath(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		bytes.NewReader([]byte(`{"name":"test-proj"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectCreate_InvalidPath covers invalid path (no .git directory).
func TestGolden_ProjectCreate_InvalidPath(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		bytes.NewReader([]byte(`{"name":"test-proj","path":"/tmp/no-git-here"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
