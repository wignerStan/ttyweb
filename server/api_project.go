package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ttyweb/service"
)

// handleProjects handles GET (list) and POST (create) for projects.
func (server *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		projects, err := projectService().ListProjects()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list projects: "+err.Error())
			return
		}
		writeAPISuccess(w, projects)

	case http.MethodPost:
		var body struct {
			Name             string `json:"name"`
			Path             string `json:"path"`
			Description      string `json:"description"`
			WorktreeBasePath string `json:"worktree_base_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var opts []service.ProjectOption
		if body.Description != "" {
			opts = append(opts, service.WithDescription(body.Description))
		}
		if body.WorktreeBasePath != "" {
			opts = append(opts, service.WithWorktreeBasePath(body.WorktreeBasePath))
		}

		project, err := projectService().AddProject(body.Name, body.Path, opts...)
		if err != nil {
			writeProjectError(w, err, "create")
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeAPISuccess(w, project)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProjectDetail handles GET, PUT, DELETE for a specific project,
// as well as POST /api/projects/:id/sync.
func (server *Server) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	prefix := server.options.Path + "api/projects/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "project ID required")
		return
	}

	// Route: /api/projects/{id}/sync
	id, isSync := strings.CutSuffix(relative, "/sync")
	if isSync {
		project, err := projectService().SyncProject(id)
		if err != nil {
			writeProjectError(w, err, "sync")
			return
		}
		writeAPISuccess(w, project)
		return
	}

	switch r.Method {
	case http.MethodGet:
		project, err := projectService().GetProject(relative)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "project not found")
			return
		}
		writeAPISuccess(w, project)

	case http.MethodPut:
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		project, err := projectService().UpdateProject(relative, body)
		if err != nil {
			writeProjectError(w, err, "update")
			return
		}
		writeAPISuccess(w, project)

	case http.MethodDelete:
		err := projectService().DeleteProject(relative)
		if err != nil {
			writeProjectError(w, err, "delete")
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// writeProjectError maps service errors to appropriate HTTP responses.
func writeProjectError(w http.ResponseWriter, err error, action string) {
	if errors.Is(err, service.ErrProjectNotFound) {
		writeAPIError(w, http.StatusNotFound, "project not found")
		return
	}
	switch err {
	case service.ErrProjectNameRequired:
		writeAPIError(w, http.StatusBadRequest, "name is required")
	case service.ErrProjectPathRequired:
		writeAPIError(w, http.StatusBadRequest, "path is required")
	case service.ErrInvalidProjectPath:
		writeAPIError(w, http.StatusBadRequest, "path must be an absolute path to a directory containing .git")
	case service.ErrProjectAlreadyExists:
		writeAPIError(w, http.StatusConflict, "a project with this path already exists")
	default:
		writeAPIError(w, http.StatusInternalServerError, "failed to "+action+" project: "+err.Error())
	}
}
