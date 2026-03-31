package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlePRCheckout_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1/pr-checkout", nil)
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandlePRCheckout_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePRCheckout_MissingPRNumber(t *testing.T) {
	body := `{"github_token": "tok"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePRCheckout_ZeroPRNumber(t *testing.T) {
	body := `{"pr_number": 0, "github_token": "tok"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePRCheckout_MissingToken(t *testing.T) {
	body := `{"pr_number": 1}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePRCheckout_EmptyToken(t *testing.T) {
	body := `{"pr_number": 1, "github_token": "   "}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlePRCheckout_ProjectNotFound(t *testing.T) {
	body := `{"pr_number": 1, "github_token": "tok"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/nonexistent/pr-checkout", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "nonexistent")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Success {
		t.Fatal("expected failure")
	}
	if resp.Error != "project not found" {
		t.Errorf("error = %q, want %q", resp.Error, "project not found")
	}
}

func TestHandlePRCheckout_EmptyBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	handlePRCheckout(rec, req, "p1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestMakeProjectDetailHandler_PRCheckoutRoute(t *testing.T) {
	handler := makeProjectDetailHandler("/api/worktree/projects")
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/pr-checkout", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
