package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleGroups handles GET (list) and POST (create) for session groups.
func (server *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		profileKey := r.URL.Query().Get("profile_key")
		groups := store.ListGroups(profileKey)
		writeAPISuccess(w, groups)

	case http.MethodPost:
		var body struct {
			GroupName  string `json:"group_name"`
			SortOrder  int    `json:"sort_order"`
			ProfileKey string `json:"profile_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.GroupName == "" {
			writeAPIError(w, http.StatusBadRequest, "group_name is required")
			return
		}
		created := store.CreateGroup(SessionGroup{
			GroupName:  body.GroupName,
			SortOrder:  body.SortOrder,
			ProfileKey: body.ProfileKey,
		})
		writeAPISuccess(w, created)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGroupDetail handles PUT (update) and DELETE for a specific group by ID.
func (server *Server) handleGroupDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path: /api/groups/{id}
	prefix := server.options.Path + "api/groups/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "group ID required")
		return
	}

	id, err := strconv.Atoi(relative)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid group ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var body struct {
			GroupName  string `json:"group_name"`
			SortOrder  int    `json:"sort_order"`
			ProfileKey string `json:"profile_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		updated, err := store.UpdateGroup(id, SessionGroup{
			GroupName:  body.GroupName,
			SortOrder:  body.SortOrder,
			ProfileKey: body.ProfileKey,
		})
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, updated)

	case http.MethodDelete:
		if err := store.DeleteGroup(id); err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
