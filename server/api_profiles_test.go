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

func TestHandleProfiles_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/profiles", nil)
	srv.handleProfiles(rec, req)

	res := rec.Result()
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var resp apiResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}
}

func TestHandleProfiles_POST(t *testing.T) {
	srv := newTestServer()
	body := `{"profile_key":"pk1","name":"Profile 1","sort_order":1}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfiles(rec, req)

	res := rec.Result()
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", res.StatusCode, rec.Body.String())
	}

	var resp struct {
		Success bool    `json:"success"`
		Data    Profile `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.ID != 1 {
		t.Fatalf("expected ID 1, got %d", resp.Data.ID)
	}
	if resp.Data.ProfileKey != "pk1" {
		t.Fatalf("expected profile_key pk1, got %q", resp.Data.ProfileKey)
	}
}

func TestHandleProfiles_POST_MissingProfileKey(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"Profile 1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfiles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProfiles_POST_MissingName(t *testing.T) {
	srv := newTestServer()
	body := `{"profile_key":"pk1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfiles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProfiles_POST_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfiles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProfiles_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/profiles", nil)
	srv.handleProfiles(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleProfileDetail_PUT(t *testing.T) {
	srv := newTestServer()
	// Create a profile first.
	created := store.CreateProfile(Profile{ProfileKey: "pk-update", Name: "Old"})
	body := `{"profile_key":"pk-update","name":"Updated","sort_order":5}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, fmt.Sprintf("/api/profiles/%d", created.ID), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool    `json:"success"`
		Data    Profile `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, resp.Data.ID)
	}
	if resp.Data.Name != "Updated" {
		t.Fatalf("expected Updated, got %q", resp.Data.Name)
	}
}

func TestHandleProfileDetail_PUT_NotFound(t *testing.T) {
	srv := newTestServer()
	body := `{"profile_key":"pk1","name":"X"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/profiles/999", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleProfileDetail_DELETE(t *testing.T) {
	srv := newTestServer()
	created := store.CreateProfile(Profile{ProfileKey: "pk-delete", Name: "A"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/api/profiles/%d", created.ID), nil)
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify deletion.
	for _, p := range store.ListProfiles() {
		if p.ID == created.ID {
			t.Fatal("profile not deleted")
		}
	}
}

func TestHandleProfileDetail_DELETE_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/profiles/999", nil)
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleProfileDetail_InvalidID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/profiles/abc", nil)
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProfileDetail_EmptyPath(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/profiles/", nil)
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProfileDetail_PUT_InvalidBody(t *testing.T) {
	srv := newTestServer()
	store.CreateProfile(Profile{Name: "TestProfile", ProfileKey: "bash"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/profiles/1", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleProfileDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/profiles/1", nil)
	srv.handleProfileDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
