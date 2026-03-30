package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"ttyweb/config"
	"ttyweb/service"
	"ttyweb/worktree"
)

// commitMsgService is the global commit message service instance.
var commitMsgService = service.NewCommitMessageService(config.Get())

// handleAICommitMessage generates an AI-powered commit message for a worktree.
// POST /api/worktree/projects/{projectID}/worktrees/{worktreeID}/ai-commit-message
func handleAICommitMessage(w http.ResponseWriter, r *http.Request, projectID, worktreeID string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate project exists and get path.
	project, err := wtService.GetProject(projectID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "project not found")
		return
	}

	// Look up the worktree record to get its filesystem path.
	record, err := findWorktreeRecord(r.Context(), projectID, worktreeID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get the diff for the worktree.
	diff, err := worktree.GetWorktreeDiff(r.Context(), project.Path, record.Path)
	if err != nil {
		slog.Error("failed to get worktree diff", "error", err, "worktree", worktreeID)
		writeAPIError(w, http.StatusBadRequest, "failed to get worktree diff")
		return
	}

	if diff == "" {
		writeAPIError(w, http.StatusBadRequest, "no changes detected in worktree")
		return
	}

	// Generate commit message via LLM.
	message, err := commitMsgService.GenerateCommitMessage(r.Context(), diff)
	if err != nil {
		slog.Error("failed to generate commit message", "error", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to generate commit message: "+err.Error())
		return
	}

	writeAPISuccess(w, map[string]string{"message": message})
}

// findWorktreeRecord looks up a worktree record by project and worktree ID.
func findWorktreeRecord(ctx context.Context, projectID, worktreeID string) (*service.WorktreeRecord, error) {
	worktrees, err := wtService.ListWorktrees(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for i := range worktrees {
		if worktrees[i].ID == worktreeID {
			return &worktrees[i], nil
		}
	}
	return nil, fmt.Errorf("worktree not found: %s", worktreeID)
}
