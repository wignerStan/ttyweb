package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"ttyweb/service"
)

// handleProjects handles GET (list) and POST (create) for projects.
func (*Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	svc, err := projectService()
	if err != nil {
		log.Printf("failed to initialize project service: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to initialize project service")
		return
	}

	switch r.Method {
	case http.MethodGet:
		projects, err := svc.ListProjects()
		if err != nil {
			log.Printf("failed to list projects: %v", err)
			writeAPIError(w, http.StatusInternalServerError, "failed to list projects")
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

		project, err := svc.AddProject(body.Name, body.Path, opts...)
		if err != nil {
			switch err {
			case service.ErrProjectNameRequired:
				writeAPIError(w, http.StatusBadRequest, "name is required")
				return
			case service.ErrProjectPathRequired:
				writeAPIError(w, http.StatusBadRequest, "path is required")
				return
			case service.ErrInvalidProjectPath:
				writeAPIError(w, http.StatusBadRequest, "path must be an absolute path to a directory containing .git")
				return
			case service.ErrProjectAlreadyExists:
				writeAPIError(w, http.StatusConflict, "a project with this path already exists")
				return
			default:
				log.Printf("failed to create project: %v", err)
				writeAPIError(w, http.StatusInternalServerError, "failed to create project")
				return
			}
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
	svc, err := projectService()
	if err != nil {
		log.Printf("failed to initialize project service: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to initialize project service")
		return
	}

	prefix := server.options.Path + "api/projects/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "project ID required")
		return
	}

	// Route: /api/projects/{id}/sync
	if id, ok := strings.CutSuffix(relative, "/sync"); ok {
		server.handleProjectSync(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		project, err := svc.GetProject(relative)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "project not found")
			return
		}
		writeAPISuccess(w, project)

	case http.MethodPut:
		var body service.UpdateProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		project, err := svc.UpdateProject(relative, body)
		if err != nil {
			if errors.Is(err, service.ErrProjectNotFound) {
				writeAPIError(w, http.StatusNotFound, "project not found")
				return
			}
			log.Printf("failed to update project: %v", err)
			writeAPIError(w, http.StatusInternalServerError, "failed to update project")
			return
		}
		writeAPISuccess(w, project)

	case http.MethodDelete:
		err := svc.DeleteProject(relative)
		if err != nil {
			if errors.Is(err, service.ErrProjectNotFound) {
				writeAPIError(w, http.StatusNotFound, "project not found")
				return
			}
			log.Printf("failed to delete project: %v", err)
			writeAPIError(w, http.StatusInternalServerError, "failed to delete project")
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProjectSync handles POST /api/projects/:id/sync.
func (*Server) handleProjectSync(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	svc, err := projectService()
	if err != nil {
		log.Printf("failed to initialize project service: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to initialize project service")
		return
	}

	project, err := svc.SyncProject(id)
	if err != nil {
		if errors.Is(err, service.ErrProjectNotFound) {
			writeAPIError(w, http.StatusNotFound, "project not found")
			return
		}
		log.Printf("failed to sync project: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to sync project")
		return
	}

	writeAPISuccess(w, project)
}
