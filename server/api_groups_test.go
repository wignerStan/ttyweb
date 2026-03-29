package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGroups_GET_Empty(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/groups", nil)
	srv.handleGroups(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
}

func TestHandleGroups_GET_WithFilter(t *testing.T) {
	srv := newTestServer()
	store.CreateGroup(SessionGroup{GroupName: "G1", ProfileKey: "pk1"})
	store.CreateGroup(SessionGroup{GroupName: "G2", ProfileKey: "pk2"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/groups?profile_key=pk1", nil)
	srv.handleGroups(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    []SessionGroup `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Data))
	}
	if resp.Data[0].GroupName != "G1" {
		t.Fatalf("expected G1, got %q", resp.Data[0].GroupName)
	}
}

func TestHandleGroups_POST(t *testing.T) {
	srv := newTestServer()
	body := `{"group_name":"G1","sort_order":1,"profile_key":"pk1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/groups", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleGroups(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    SessionGroup `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.ID < 1 {
		t.Fatalf("expected positive ID, got %d", resp.Data.ID)
	}
}

func TestHandleGroups_POST_MissingGroupName(t *testing.T) {
	srv := newTestServer()
	body := `{"sort_order":1}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/groups", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleGroups(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGroups_POST_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/groups", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleGroups(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGroups_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/groups", nil)
	srv.handleGroups(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleGroupDetail_PUT(t *testing.T) {
	srv := newTestServer()
	created := store.CreateGroup(SessionGroup{GroupName: "G1"})
	body := `{"group_name":"G1-updated","sort_order":5,"profile_key":"pk1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, fmt.Sprintf("/api/groups/%d", created.ID), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool         `json:"success"`
		Data    SessionGroup `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.GroupName != "G1-updated" {
		t.Fatalf("expected G1-updated, got %q", resp.Data.GroupName)
	}
	_ = created
}

func TestHandleGroupDetail_PUT_NotFound(t *testing.T) {
	srv := newTestServer()
	body := `{"group_name":"X"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/groups/999", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGroupDetail_DELETE(t *testing.T) {
	srv := newTestServer()
	created := store.CreateGroup(SessionGroup{GroupName: "G1-Delete"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/api/groups/%d", created.ID), nil)
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Verify deletion.
	for _, g := range store.ListGroups("") {
		if g.ID == created.ID {
			t.Fatal("group not deleted")
		}
	}
}

func TestHandleGroupDetail_DELETE_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/groups/999", nil)
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGroupDetail_InvalidID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/groups/abc", nil)
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGroupDetail_EmptyPath(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/groups/", nil)
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGroupDetail_PUT_InvalidBody(t *testing.T) {
	srv := newTestServer()
	store.CreateGroup(SessionGroup{GroupName: "TestGroup", ProfileKey: "default"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/groups/1", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGroupDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/groups/1", nil)
	srv.handleGroupDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
