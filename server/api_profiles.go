package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleProfiles handles GET (list) and POST (create) for profiles.
func (*Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		profiles := store.ListProfiles()
		writeAPISuccess(w, profiles)

	case http.MethodPost:
		var body struct {
			ProfileKey string `json:"profile_key"`
			Name       string `json:"name"`
			SortOrder  int    `json:"sort_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.ProfileKey == "" {
			writeAPIError(w, http.StatusBadRequest, "profile_key is required")
			return
		}
		if body.Name == "" {
			writeAPIError(w, http.StatusBadRequest, "name is required")
			return
		}
		created := store.CreateProfile(Profile{
			ProfileKey: body.ProfileKey,
			Name:       body.Name,
			SortOrder:  body.SortOrder,
		})
		writeAPISuccess(w, created)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProfileDetail handles PUT (update) and DELETE for a specific profile by ID.
func (server *Server) handleProfileDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path: /api/profiles/{id}
	prefix := server.options.Path + "api/profiles/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "profile ID required")
		return
	}

	id, err := strconv.Atoi(relative)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid profile ID")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var body struct {
			ProfileKey string `json:"profile_key"`
			Name       string `json:"name"`
			SortOrder  int    `json:"sort_order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		updated, err := store.UpdateProfile(id, Profile{
			ProfileKey: body.ProfileKey,
			Name:       body.Name,
			SortOrder:  body.SortOrder,
		})
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, updated)

	case http.MethodDelete:
		if err := store.DeleteProfile(id); err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
