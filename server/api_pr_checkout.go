package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"ttyweb/service"
)

// prCheckoutService is the global PR checkout service instance.
var prCheckoutService = service.NewPRCheckoutService()

// handlePRCheckout handles POST requests to check out a pull request branch.
// Request body: { "pr_number": int, "github_token": string, "repo_slug": string? }
func handlePRCheckout(w http.ResponseWriter, r *http.Request, projectID string) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		PRNumber    int    `json:"pr_number"`
		GitHubToken string `json:"github_token"`
		RepoSlug    string `json:"repo_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.PRNumber <= 0 {
		writeAPIError(w, http.StatusBadRequest, "pr_number is required and must be positive")
		return
	}
	if strings.TrimSpace(body.GitHubToken) == "" {
		writeAPIError(w, http.StatusBadRequest, "github_token is required")
		return
	}

	project, err := wtService.GetProject(projectID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "project not found")
		return
	}

	worktreePath, err := prCheckoutService.CheckoutPR(r.Context(), project.Path, body.RepoSlug, body.PRNumber, body.GitHubToken)
	if err != nil {
		slog.Error("PR checkout failed", "project_id", projectID, "pr_number", body.PRNumber, "error", err)
		writeAPIError(w, http.StatusBadRequest, "PR checkout failed: "+err.Error())
		return
	}

	writeAPISuccess(w, map[string]string{
		"path":   worktreePath,
		"branch": fmt.Sprintf("pr-%d", body.PRNumber),
	})
}
