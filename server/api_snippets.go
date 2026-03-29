package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// handleSnippets handles GET (list) and POST (create) for snippets.
func (_ *Server) handleSnippets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		snippets := store.ListSnippets()
		writeAPISuccess(w, snippets)

	case http.MethodPost:
		var body struct {
			Name    string `json:"name"`
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.Name == "" {
			writeAPIError(w, http.StatusBadRequest, "name is required")
			return
		}
		if body.Command == "" {
			writeAPIError(w, http.StatusBadRequest, "command is required")
			return
		}
		created := store.CreateSnippet(Snippet{
			Name:    body.Name,
			Command: body.Command,
		})
		writeAPISuccess(w, created)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSnippetDetail handles PUT (update) and DELETE for a specific snippet by index.
func (server *Server) handleSnippetDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract index from path: /api/snippets/{index}
	prefix := server.options.Path + "api/snippets/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "snippet index required")
		return
	}

	index, err := strconv.Atoi(relative)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid snippet index")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var body struct {
			Name    string `json:"name"`
			Command string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		updated, err := store.UpdateSnippet(index, Snippet{
			Name:    body.Name,
			Command: body.Command,
		})
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, updated)

	case http.MethodDelete:
		if err := store.DeleteSnippet(index); err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
