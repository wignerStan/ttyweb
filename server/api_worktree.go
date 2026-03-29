package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"ttyweb/service"
)

// worktreeService is the global worktree service instance.
var wtService = service.NewWorktreeService()

// setupWorktreeRoutes registers worktree API routes on the given mux.
func setupWorktreeRoutes(mux *http.ServeMux, apiPrefix string) {
	projectPrefix := apiPrefix + "worktree/projects"
	mux.HandleFunc(projectPrefix, handleWorktreeProjects)
	mux.HandleFunc(projectPrefix+"/", makeProjectDetailHandler(projectPrefix))
}

func makeProjectDetailHandler(projectPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := strings.TrimPrefix(r.URL.Path, projectPrefix+"/")
		path = strings.TrimSuffix(path, "/")

		parts := strings.SplitN(path, "/", 2)
		projectID := parts[0]
		if projectID == "" {
			writeAPIError(w, http.StatusBadRequest, "project ID required")
			return
		}

		subPath := ""
		if len(parts) > 1 {
			subPath = parts[1]
		}

		switch {
		case subPath == "":
			writeAPIError(w, http.StatusNotFound, "not found")
		case subPath == "worktrees":
			handleWorktreeList(w, r, projectID)
		case subPath == "worktrees/sync":
			handleWorktreeSync(w, r, projectID)
		case strings.HasPrefix(subPath, "worktrees/"):
			rest := strings.TrimPrefix(subPath, "worktrees/")
			handleWorktreeItem(w, r, projectID, rest)
		}
	}
}

func handleWorktreeProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		projects := wtService.ListProjects()
		writeAPISuccess(w, projects)

	case http.MethodPost:
		var body struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if strings.TrimSpace(body.Path) == "" {
			writeAPIError(w, http.StatusBadRequest, "path is required")
			return
		}
		project, err := wtService.AddProject(body.Path)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, project)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorktreeList(w http.ResponseWriter, r *http.Request, projectID string) {
	switch r.Method {
	case http.MethodGet:
		worktrees, err := wtService.ListWorktrees(projectID)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, worktrees)

	case http.MethodPost:
		var body struct {
			BranchName   string `json:"branchName"`
			BaseBranch   string `json:"baseBranch"`
			CreateBranch bool   `json:"createBranch"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if strings.TrimSpace(body.BranchName) == "" {
			writeAPIError(w, http.StatusBadRequest, "branchName is required")
			return
		}

		record, err := wtService.CreateWorktree(r.Context(), projectID, body.BranchName, body.BaseBranch, body.CreateBranch)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, record)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorktreeSync(w http.ResponseWriter, r *http.Request, projectID string) {
	switch r.Method {
	case http.MethodPost:
		worktrees, err := wtService.SyncAllWorktrees(projectID)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, worktrees)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleWorktreeItem(w http.ResponseWriter, r *http.Request, projectID, worktreePath string) {
	// worktreePath is either:
	//   {wtId}
	//   {wtId}/refresh
	//   {wtId}/commit

	switch r.Method {
	case http.MethodDelete:
		if err := wtService.RemoveWorktree(r.Context(), projectID, worktreePath, false); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "removed"})

	case http.MethodPost:
		// Parse action suffix.
		parts := strings.SplitN(worktreePath, "/", 2)
		worktreeID := parts[0]
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}

		switch action {
		case "refresh":
			record, err := wtService.RefreshWorktree(projectID, worktreeID)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeAPISuccess(w, record)

		case "commit":
			var body struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if strings.TrimSpace(body.Message) == "" {
				writeAPIError(w, http.StatusBadRequest, "commit message is required")
				return
			}
			record, err := wtService.CommitWorktree(r.Context(), projectID, worktreeID, body.Message)
			if err != nil {
				writeAPIError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeAPISuccess(w, record)

		default:
			writeAPIError(w, http.StatusBadRequest, "unknown action: "+action)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
