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

func TestHandleSegments_GET_Empty(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments", nil)
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSegments_GET_WithFilter(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments?session=my-session", nil)
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleSegments_POST(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"session_name":"sess1","window_name":"win1","pane_index":0,"task_title":"Test Task"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSegments_POST_MissingTaskTitle(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"session_name":"sess1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegments_POST_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegments_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/segments", nil)
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_EmptyID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_GET_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/nonexistent-id", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_GET_Detail_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/nonexistent-id/detail", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_Detail_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/some-id/detail", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_GET_Messages_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/nonexistent-id/messages", nil)
	srv.handleSegmentDetail(rec, req)

	// Service returns empty list for nonexistent segment.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_Messages_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/some-id/messages", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_GET_Commands_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/nonexistent-id/commands", nil)
	srv.handleSegmentDetail(rec, req)

	// Service returns empty list for nonexistent segment.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_Commands_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments/some-id/commands", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_PATCH_InvalidBody(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/segments/some-id", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_PATCH_MissingFields(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/segments/some-id", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_PATCH_NotFound(t *testing.T) {
	srv := newTestServerWithDB(t)
	body := `{"task_title":"Updated"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/segments/nonexistent-id", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/segments/some-id", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestSplitSubRoute(t *testing.T) {
	tests := []struct {
		path string
		id   string
		sub  string
		ok   bool
	}{
		{"abc123/detail", "abc123", "detail", true},
		{"abc/messages", "abc", "messages", true},
		{"simple", "", "", false},
	}
	for _, tt := range tests {
		id, sub, ok := splitSubRoute(tt.path)
		if id != tt.id || sub != tt.sub || ok != tt.ok {
			t.Errorf("splitSubRoute(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.path, id, sub, ok, tt.id, tt.sub, tt.ok)
		}
	}
}

func TestHandleSegments_GET_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	// Close the DB to trigger errors.
	_ = db.Close()
	defer func() {
		// Re-init DB for other tests.
		_ = db.Init(t.TempDir() + "/reinit.db")
	}()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments", nil)
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSegments_POST_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	_ = db.Close()
	defer func() {
		_ = db.Init(t.TempDir() + "/reinit2.db")
	}()

	body := `{"task_title":"Test"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleSegments(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSegmentDetail_PATCH_Success(t *testing.T) {
	srv := newTestServerWithDB(t)
	// First create a segment.
	body := `{"session_name":"sess1","window_name":"win1","pane_index":0,"task_title":"Original Title"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleSegments(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRec.Code)
	}

	// Now get the segment to find its ID.
	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments", nil)
	srv.handleSegments(listRec, listReq)

	type segData struct {
		ID string `json:"id"`
	}
	var listResp struct {
		Success bool      `json:"success"`
		Data    []segData `json:"data"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode list: %v", err)
	}
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least one segment")
	}

	segID := listResp.Data[0].ID

	// PATCH update.
	patchBody := `{"task_title":"Updated Title"}`
	patchRec := httptest.NewRecorder()
	patchReq := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/segments/"+segID, bytes.NewReader([]byte(patchBody)))
	patchReq.Header.Set("Content-Type", "application/json")
	srv.handleSegmentDetail(patchRec, patchReq)

	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", patchRec.Code, patchRec.Body.String())
	}
}

func TestHandleSegmentDetail_PATCH_Status(t *testing.T) {
	srv := newTestServerWithDB(t)
	// First create a segment.
	body := `{"session_name":"sess1","window_name":"win1","pane_index":0,"task_title":"Status Test"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleSegments(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRec.Code)
	}

	// Get segment ID.
	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments", nil)
	srv.handleSegments(listRec, listReq)
	type segData struct {
		ID string `json:"id"`
	}
	var listResp struct {
		Success bool      `json:"success"`
		Data    []segData `json:"data"`
	}
	_ = json.NewDecoder(listRec.Body).Decode(&listResp)
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least one segment")
	}
	segID := listResp.Data[0].ID

	patchBody := `{"task_status":"in_progress"}`
	patchRec := httptest.NewRecorder()
	patchReq := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/segments/"+segID, bytes.NewReader([]byte(patchBody)))
	patchReq.Header.Set("Content-Type", "application/json")
	srv.handleSegmentDetail(patchRec, patchReq)

	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", patchRec.Code, patchRec.Body.String())
	}
}

func TestHandleSegmentDetail_GET_Detail_Success(t *testing.T) {
	srv := newTestServerWithDB(t)
	// Create a segment first.
	body := `{"session_name":"sess1","window_name":"win1","pane_index":0,"task_title":"Detail Test"}`
	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/segments", bytes.NewReader([]byte(body)))
	createReq.Header.Set("Content-Type", "application/json")
	srv.handleSegments(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on create, got %d", createRec.Code)
	}

	// Get segment ID.
	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments", nil)
	srv.handleSegments(listRec, listReq)
	type segData struct {
		ID string `json:"id"`
	}
	var listResp struct {
		Success bool      `json:"success"`
		Data    []segData `json:"data"`
	}
	_ = json.NewDecoder(listRec.Body).Decode(&listResp)
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least one segment")
	}
	segID := listResp.Data[0].ID

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/"+segID+"/detail", nil)
	srv.handleSegmentDetail(detailRec, detailReq)

	if detailRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", detailRec.Code, detailRec.Body.String())
	}
}

func TestHandleSegmentDetail_Detail_EmptyID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments//detail", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_Messages_EmptyID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments//messages", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_Commands_EmptyID(t *testing.T) {
	srv := newTestServerWithDB(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments//commands", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSegmentDetail_Messages_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	_ = db.Close()
	defer func() { _ = db.Init(t.TempDir() + "/reinit-seg1.db") }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/some-id/messages", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSegmentDetail_Commands_ClosedDB(t *testing.T) {
	srv := newTestServerWithDB(t)
	_ = db.Close()
	defer func() { _ = db.Init(t.TempDir() + "/reinit-seg2.db") }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/segments/some-id/commands", nil)
	srv.handleSegmentDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestSplitSubRoute_EmptyPath(t *testing.T) {
	id, sub, ok := splitSubRoute("")
	if ok {
		t.Fatal("expected ok=false for empty path")
	}
	if id != "" || sub != "" {
		t.Fatalf("expected empty strings, got %q %q", id, sub)
	}
}
