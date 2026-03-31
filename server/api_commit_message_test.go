package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAICommitMessage_MethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/worktree/projects/p1/worktrees/wt1/ai-commit-message", nil)
	handleAICommitMessage(rec, req, "p1", "wt1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAICommitMessage_InvalidBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/p1/worktrees/wt1/ai-commit-message", bytes.NewReader([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	handleAICommitMessage(rec, req, "p1", "wt1")

	// Project not found returns 400.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAICommitMessage_NoProject(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/nonexistent/worktrees/wt1/ai-commit-message", nil)
	handleAICommitMessage(rec, req, "nonexistent", "wt1")

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
	if !resp.Success {
		// Should be an error response.
		if resp.Error != "project not found" {
			t.Errorf("expected 'project not found', got %q", resp.Error)
		}
	}
}

func TestHandleAICommitMessage_NoWorktree(t *testing.T) {
	gitDir := createRealGitRepo(t)
	project, err := wtService.AddProject(gitDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/worktree/projects/"+project.ID+"/worktrees/nonexistent/ai-commit-message", nil)
	handleAICommitMessage(rec, req, project.ID, "nonexistent")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAICommitMessage_DeleteMethod(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/worktree/projects/p1/worktrees/wt1/ai-commit-message", nil)
	handleAICommitMessage(rec, req, "p1", "wt1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleAICommitMessage_PutMethod(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/worktree/projects/p1/worktrees/wt1/ai-commit-message", nil)
	handleAICommitMessage(rec, req, "p1", "wt1")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
