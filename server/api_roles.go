package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleRoles handles GET (list) and POST (create) for AI roles.
func (*Server) handleRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		roles := store.ListRoles()
		writeAPISuccess(w, roles)

	case http.MethodPost:
		var body struct {
			Name         string `json:"name"`
			Description  string `json:"description"`
			SystemPrompt string `json:"system_prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.Name == "" {
			writeAPIError(w, http.StatusBadRequest, "name is required")
			return
		}
		if body.SystemPrompt == "" {
			writeAPIError(w, http.StatusBadRequest, "system_prompt is required")
			return
		}
		created := store.CreateRole(AiRole{
			Name:         body.Name,
			Description:  body.Description,
			SystemPrompt: body.SystemPrompt,
		})
		writeAPISuccess(w, created)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRoleDetail handles PUT (update) and DELETE for a specific role by ID.
func (server *Server) handleRoleDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path: /api/roles/{id}
	prefix := server.options.Path + "api/roles/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "role ID required")
		return
	}

	id, err := strconv.Atoi(relative)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var body struct {
			Name         string `json:"name"`
			Description  string `json:"description"`
			SystemPrompt string `json:"system_prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		updated, err := store.UpdateRole(id, AiRole{
			Name:         body.Name,
			Description:  body.Description,
			SystemPrompt: body.SystemPrompt,
		})
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, updated)

	case http.MethodDelete:
		if err := store.DeleteRole(id); err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDefaultRoles returns the built-in default AI roles.
func (*Server) handleDefaultRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeAPISuccess(w, builtinRoles())
}
