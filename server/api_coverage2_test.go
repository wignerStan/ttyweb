package server

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGolden_TaskStats_MethodNotAllowed covers non-GET on stats endpoint.
func TestGolden_TaskStats_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/tasks/stats", nil,
	)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_TaskStats_NoService covers the nil statsService path.
func TestGolden_TaskStats_NoService(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/tasks/stats?days=30", nil,
	)
	srv.handleTaskStats(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_Upload_MethodNotAllowed covers non-POST on upload endpoint.
func TestGolden_Upload_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/upload", nil,
	)
	srv.handleUpload(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_Upload_NoFile covers upload without a file field.
func TestGolden_Upload_NoFile(t *testing.T) {
	srv := newTestServer()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/upload", &buf,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	srv.handleUpload(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_Upload_Success covers successful file upload.
func TestGolden_Upload_Success(t *testing.T) {
	srv := newTestServer()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("hello world")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = writer.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/upload", &buf,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	srv.handleUpload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestGolden_PRCheckout_MethodNotAllowed covers non-POST on PR checkout.
func TestGolden_PRCheckout_MethodNotAllowed(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/projects/proj-1/pr-checkout", nil,
	)
	handlePRCheckout(rec, req, "proj-1")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_PRCheckout_BadJSON covers invalid JSON body.
func TestGolden_PRCheckout_BadJSON(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects/proj-1/pr-checkout",
		bytes.NewReader([]byte("not json")),
	)
	handlePRCheckout(rec, req, "proj-1")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_PRCheckout_MissingFields covers missing required fields.
func TestGolden_PRCheckout_MissingFields(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects/proj-1/pr-checkout",
		bytes.NewReader([]byte(`{"pr_number":0}`)),
	)
	handlePRCheckout(rec, req, "proj-1")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_PRCheckout_NoToken covers missing token.
func TestGolden_PRCheckout_NoToken(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects/proj-1/pr-checkout",
		bytes.NewReader([]byte(`{"pr_number":1,"github_token":"   "}`)),
	)
	handlePRCheckout(rec, req, "proj-1")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_TaskEventsPane_MethodNotAllowed covers non-GET on events pane.
func TestGolden_TaskEventsPane_MethodNotAllowed(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/tasks/events/pane-1", nil,
	)
	handleTaskEventsPane(rec, req, "pane-1")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_TaskEventsPane_EmptyPane covers empty pane key.
func TestGolden_TaskEventsPane_EmptyPane(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/tasks/events/", nil,
	)
	handleTaskEventsPane(rec, req, "")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_TaskEventsPane_Success covers the success path.
func TestGolden_TaskEventsPane_Success(t *testing.T) {
	store = NewMemoryStore()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet,
		"/api/tasks/events/pane-1", nil,
	)
	handleTaskEventsPane(rec, req, "pane-1")

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_ProjectCreate_AlreadyExists covers duplicate project path.
func TestGolden_ProjectCreate_AlreadyExists(t *testing.T) {
	srv := newTestServerWithDB(t)

	projectDir := t.TempDir()
	mustGit(t, projectDir, "init", "-b", "main")
	mustGit(t, projectDir, "config", "user.name", "test")
	mustGit(t, projectDir, "config", "user.email", "test@test.com")

	// Create first project.
	createBody := bytes.NewReader([]byte(`{"name":"dup-proj","path":"` + projectDir + `"}`))
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		createBody,
	)
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleProjects(createRec, createReq)

	// Try to create second project with same path.
	createBody2 := bytes.NewReader([]byte(`{"name":"dup-proj-2","path":"` + projectDir + `"}`))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/projects",
		createBody2,
	)
	req.Header.Set("Content-Type", "application/json")
	srv.handleProjects(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
