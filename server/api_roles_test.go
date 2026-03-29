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

func TestHandleRoles_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/roles", nil)
	srv.handleRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool     `json:"success"`
		Data    []AiRole `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", "")
	}
	if len(resp.Data) != 7 {
		t.Fatalf("expected 7 builtin roles, got %d", len(resp.Data))
	}
}

func TestHandleRoles_POST(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"Custom","description":"A custom role","system_prompt":"Be helpful"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Data    AiRole `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.ID != 8 {
		t.Fatalf("expected ID 8, got %d", resp.Data.ID)
	}
	if resp.Data.Name != "Custom" {
		t.Fatalf("expected 'Custom', got %q", resp.Data.Name)
	}
}

func TestHandleRoles_POST_MissingName(t *testing.T) {
	srv := newTestServer()
	body := `{"system_prompt":"Be helpful"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRoles_POST_MissingSystemPrompt(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"Custom"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/roles", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRoles_POST_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/roles", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoles(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRoles_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/roles", nil)
	srv.handleRoles(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleRoleDetail_PUT(t *testing.T) {
	srv := newTestServer()
	created := store.CreateRole(AiRole{Name: "Custom-Update", SystemPrompt: "Old prompt"})
	body := `{"name":"Updated","description":"Updated desc","system_prompt":"New prompt"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, fmt.Sprintf("/api/roles/%d", created.ID), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool   `json:"success"`
		Data    AiRole `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, resp.Data.ID)
	}
	if resp.Data.SystemPrompt != "New prompt" {
		t.Fatalf("expected 'New prompt', got %q", resp.Data.SystemPrompt)
	}
}

func TestHandleRoleDetail_PUT_NotFound(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"X","system_prompt":"Y"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/roles/999", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRoleDetail_DELETE(t *testing.T) {
	srv := newTestServer()
	created := store.CreateRole(AiRole{Name: "Custom-Delete", SystemPrompt: "prompt"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, fmt.Sprintf("/api/roles/%d", created.ID), nil)
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// Verify deletion.
	for _, r := range store.ListRoles() {
		if r.ID == created.ID {
			t.Fatal("role not deleted")
		}
	}
}

func TestHandleRoleDetail_DELETE_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/roles/999", nil)
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRoleDetail_InvalidID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/roles/abc", nil)
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRoleDetail_EmptyPath(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/roles/", nil)
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleDefaultRoles_GET(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/roles/defaults", nil)
	srv.handleDefaultRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool     `json:"success"`
		Data    []AiRole `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 7 {
		t.Fatalf("expected 7 default roles, got %d", len(resp.Data))
	}
}

func TestHandleDefaultRoles_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/roles/defaults", nil)
	srv.handleDefaultRoles(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleRoleDetail_PUT_InvalidBody(t *testing.T) {
	srv := newTestServer()
	store.CreateRole(AiRole{Name: "TestRole", SystemPrompt: "test"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/roles/1", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleRoleDetail_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/roles/1", nil)
	srv.handleRoleDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
