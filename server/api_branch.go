package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"ttyweb/worktree"
)

// setupBranchRoutes registers branch API routes on the given mux.
func setupBranchRoutes(mux *http.ServeMux, apiPrefix string) {
	branchPrefix := apiPrefix + "branches"
	mux.HandleFunc(branchPrefix, handleBranches)
	mux.HandleFunc(branchPrefix+"/", handleBranchDetail)
}

// handleBranches dispatches GET (list) and POST (create) on /api/branches?repo=<path>.
func handleBranches(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo")
	if strings.TrimSpace(repoPath) == "" {
		writeAPIError(w, http.StatusBadRequest, "repo query parameter is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		branches, err := worktree.ListBranches(r.Context(), repoPath)
		if err != nil {
			slog.Error("failed to list branches", "repo", repoPath, "error", err)
			writeAPIError(w, http.StatusBadRequest, "failed to list branches")
			return
		}
		writeAPISuccess(w, branches)

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			writeAPIError(w, http.StatusBadRequest, "branch name is required")
			return
		}
		if err := worktree.ValidateBranchName(body.Name); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := worktree.CreateBranch(r.Context(), repoPath, body.Name); err != nil {
			slog.Error("failed to create branch", "repo", repoPath, "name", body.Name, "error", err)
			writeAPIError(w, http.StatusBadRequest, "failed to create branch")
			return
		}
		writeAPISuccess(w, map[string]string{"name": body.Name, "status": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleBranchDetail dispatches DELETE (delete) and POST merge on
// /api/branches/{name}?repo=<path> and /api/branches/merge?repo=<path>.
func handleBranchDetail(w http.ResponseWriter, r *http.Request) {
	// Extract the sub-path after /api/branches/.
	suffix := strings.TrimPrefix(r.URL.Path, "/api/branches/")
	suffix = strings.TrimPrefix(suffix, "branches/")
	suffix = strings.TrimSuffix(suffix, "/")

	repoPath := r.URL.Query().Get("repo")
	if strings.TrimSpace(repoPath) == "" {
		writeAPIError(w, http.StatusBadRequest, "repo query parameter is required")
		return
	}

	if suffix == "merge" {
		handleBranchMerge(w, r, repoPath)
		return
	}

	handleBranchDelete(w, r, repoPath, suffix)
}

// handleBranchMerge processes POST /api/branches/merge?repo=<path>.
func handleBranchMerge(w http.ResponseWriter, r *http.Request, repoPath string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Source) == "" {
		writeAPIError(w, http.StatusBadRequest, "source branch is required")
		return
	}
	if strings.TrimSpace(body.Target) == "" {
		writeAPIError(w, http.StatusBadRequest, "target branch is required")
		return
	}
	if err := worktree.MergeBranch(r.Context(), repoPath, body.Source, body.Target); err != nil {
		slog.Error("failed to merge branches", "repo", repoPath, "source", body.Source, "target", body.Target, "error", err)
		writeAPIError(w, http.StatusBadRequest, "failed to merge branches")
		return
	}
	writeAPISuccess(w, map[string]string{
		"source": body.Source,
		"target": body.Target,
		"status": "merged",
	})
}

// handleBranchDelete processes DELETE /api/branches/{name}?repo=<path>.
func handleBranchDelete(w http.ResponseWriter, r *http.Request, repoPath, branchName string) {
	if branchName == "" {
		writeAPIError(w, http.StatusBadRequest, "branch name is required")
		return
	}
	if err := worktree.ValidateBranchName(branchName); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := worktree.DeleteBranch(r.Context(), repoPath, branchName); err != nil {
		slog.Error("failed to delete branch", "repo", repoPath, "name", branchName, "error", err)
		writeAPIError(w, http.StatusBadRequest, "failed to delete branch")
		return
	}
	writeAPISuccess(w, map[string]string{"name": branchName, "status": "deleted"})
}
