package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ttyweb/db"
	"ttyweb/service"
)

// --- normalizePath ---

func TestNormalizePath_Plain(t *testing.T) {
	srv := &Server{options: &Options{Path: "app"}}
	got := srv.normalizePath()
	if got != "/app/" {
		t.Errorf("normalizePath() = %q, want %q", got, "/app/")
	}
}

func TestNormalizePath_WithSlashes(t *testing.T) {
	srv := &Server{options: &Options{Path: "/app/"}}
	got := srv.normalizePath()
	if got != "/app/" {
		t.Errorf("normalizePath() = %q, want %q", got, "/app/")
	}
}

func TestNormalizePath_Root(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	got := srv.normalizePath()
	if got != "/" {
		t.Errorf("normalizePath() = %q, want %q", got, "/")
	}
}

// --- handleConfig ---

func TestHandleConfig_NoCWD(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config", nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// When no cwd is provided, handleConfig returns {"success":true,"data":null}.
	if !strings.Contains(rec.Body.String(), `"data":null`) {
		t.Errorf("expected data:null, got %s", rec.Body.String())
	}
}

func TestHandleConfig_InvalidCWD(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config?cwd=/nonexistent", nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(configFile, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config?cwd="+dir, nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleConfig_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, ".opencode.json")
	if err := os.WriteFile(configFile, []byte(`{"key":"value"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/opencode-config?cwd="+dir, nil)
	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected non-null data for valid config file")
	}
}

// --- writeProjectCreateError ---

func TestWriteProjectCreateError_NameRequired(t *testing.T) {
	rec := httptest.NewRecorder()
	writeProjectCreateError(rec, service.ErrProjectNameRequired)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWriteProjectCreateError_PathRequired(t *testing.T) {
	rec := httptest.NewRecorder()
	writeProjectCreateError(rec, service.ErrProjectPathRequired)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWriteProjectCreateError_InvalidPath(t *testing.T) {
	rec := httptest.NewRecorder()
	writeProjectCreateError(rec, service.ErrInvalidProjectPath)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestWriteProjectCreateError_AlreadyExists(t *testing.T) {
	rec := httptest.NewRecorder()
	writeProjectCreateError(rec, service.ErrProjectAlreadyExists)
	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestWriteProjectCreateError_Default(t *testing.T) {
	rec := httptest.NewRecorder()
	writeProjectCreateError(rec, os.ErrNotExist)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- updateProject ---

func TestUpdateProject_InvalidBody(t *testing.T) {
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
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/projects/p1", strings.NewReader("not json"))
	updateProject(rec, req, svc, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", rec.Code)
	}
}

func TestUpdateProject_NotFound(t *testing.T) {
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

	body := `{"name":"updated"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/projects/nonexistent", strings.NewReader(body))
	updateProject(rec, req, svc, "nonexistent")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- listTaskAISessions with aiSessionID ---

func TestListTaskAISessions_WithSessionID(t *testing.T) {
	store = NewMemoryStore()
	rec := httptest.NewRecorder()
	listTaskAISessions(rec, "task-1", "session-1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 when aiSessionID is provided, got %d", rec.Code)
	}
}

// --- unlinkTaskAISession without sessionID ---

func TestUnlinkTaskAISession_NoSessionID(t *testing.T) {
	store = NewMemoryStore()
	rec := httptest.NewRecorder()
	unlinkTaskAISession(rec, "task-1", "")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 when aiSessionID is empty, got %d", rec.Code)
	}
}
