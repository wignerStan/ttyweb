package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ttyweb/db"
)

// extractNoteID parses the API response and returns the note ID.
func extractNoteID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	id, ok := resp.Data["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected note ID in response")
	}
	return id
}

func TestHandleNotepad_GET_Empty(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad", nil)
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
}

func TestHandleNotepad_POST(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"Test Note","content":"Hello world"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data["name"] != "Test Note" {
		t.Fatalf("expected 'Test Note', got %v", resp.Data["name"])
	}
}

func TestHandleNotepad_POST_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleNotepad_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/notepad", nil)
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleNotepadDetail_GET_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad/nonexistent-id", nil)
	srv.handleNotepadDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleNotepadDetail_EmptyID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad/", nil)
	srv.handleNotepadDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleNotepadDetail_PUT_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/notepad/some-id", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepadDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleNotepadDetail_DELETE_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/notepad/nonexistent-id", nil)
	srv.handleNotepadDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleNotepadDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/notepad/some-id", nil)
	srv.handleNotepadDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleNotepadReorder_PATCH(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `[]`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/notepad/reorder", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepadReorder(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNotepadReorder_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/notepad/reorder", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepadReorder(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleNotepadReorder_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad/reorder", nil)
	srv.handleNotepadReorder(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleNotepad_POST_WithProjectID(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"ProjNote","content":"note content","project_id":"proj-1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNotepad_GET_WithProjectFilter(t *testing.T) {
	srv := newTestServerWithDB(t)
	// Create a note with a project_id.
	body := `{"name":"FilterNote","content":"content","project_id":"proj-1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d", rec.Code)
	}

	// Filter by project_id.
	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad?project_id=proj-1", nil)
	srv.handleNotepad(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", listRec.Code, listRec.Body.String())
	}

	var listResp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(listResp.Data) != 1 {
		t.Fatalf("expected 1 note, got %d", len(listResp.Data))
	}
}

func TestHandleNotepadDetail_GET_ValidNote(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"GetNote","content":"hello"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(createRec, createReq)

	id := extractNoteID(t, createRec)

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad/"+id, nil)
	srv.handleNotepadDetail(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", getRec.Code, getRec.Body.String())
	}
}

func TestHandleNotepadDetail_PUT_ValidNote(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"UpdateNote","content":"original"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(createRec, createReq)

	id := extractNoteID(t, createRec)

	putBody := `{"content":"updated content"}`
	putRec := httptest.NewRecorder()
	putReq := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/notepad/"+id, bytes.NewReader([]byte(putBody)))
	putReq.Header.Set("Content-Type", "application/json")
	srv.handleNotepadDetail(putRec, putReq)

	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", putRec.Code, putRec.Body.String())
	}
}

func TestHandleNotepadDetail_PUT_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	putBody := `{"name":"ghost"}`
	putRec := httptest.NewRecorder()
	putReq := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/notepad/nonexistent", bytes.NewReader([]byte(putBody)))
	putReq.Header.Set("Content-Type", "application/json")
	srv.handleNotepadDetail(putRec, putReq)

	if putRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", putRec.Code)
	}
}

func TestHandleNotepadDetail_DELETE_ValidNote(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"name":"DeleteNote","content":"bye"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(createRec, createReq)

	id := extractNoteID(t, createRec)

	delRec := httptest.NewRecorder()
	delReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/notepad/"+id, nil)
	srv.handleNotepadDetail(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", delRec.Code, delRec.Body.String())
	}

	// Verify deleted.
	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad/"+id, nil)
	srv.handleNotepadDetail(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRec.Code)
	}
}

func TestHandleNotepadReorder_WithNotes(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Create two notes.
	body1 := `{"name":"NoteA","content":"a"}`
	body2 := `{"name":"NoteB","content":"b"}`
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body1)))
	req1.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec1, req1)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body2)))
	req2.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec2, req2)

	idA := extractNoteID(t, rec1)
	idB := extractNoteID(t, rec2)

	// Reorder: swap positions.
	reorderBody := `[{"id":"` + idB + `","position":0},{"id":"` + idA + `","position":1}]`
	reorderRec := httptest.NewRecorder()
	reorderReq := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/notepad/reorder", bytes.NewReader([]byte(reorderBody)))
	reorderReq.Header.Set("Content-Type", "application/json")
	srv.handleNotepadReorder(reorderRec, reorderReq)

	if reorderRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", reorderRec.Code, reorderRec.Body.String())
	}
}

func TestHandleNotepadReorder_NonexistentID(t *testing.T) {
	srv := newTestServerWithDB(t)
	reorderBody := `[{"id":"nonexistent-id","position":0}]`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/notepad/reorder", bytes.NewReader([]byte(reorderBody)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepadReorder(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleNotepad_GET_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	_ = db.Close()
	defer func() { _ = db.Init(t.TempDir() + "/reinit-n1.db") }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/notepad", nil)
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNotepad_POST_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	_ = db.Close()
	defer func() { _ = db.Init(t.TempDir() + "/reinit-n2.db") }()

	body := `{"name":"Test","content":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/notepad", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleNotepad(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleNotepadDetail_DELETE_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	_ = db.Close()
	defer func() { _ = db.Init(t.TempDir() + "/reinit-n3.db") }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/notepad/some-id", nil)
	srv.handleNotepadDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}
