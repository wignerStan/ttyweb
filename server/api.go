package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"ttyweb/backend/tmux"
	"ttyweb/pkg/validate"
)

// apiResponse is a standard envelope for API responses.
type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// setupAPIHandlers registers REST API routes on the given mux.
func (server *Server) setupAPIHandlers(mux *http.ServeMux, pathPrefix string) {
	apiPrefix := pathPrefix + "api/"
	mux.HandleFunc(apiPrefix+"sessions", server.handleListSessions)
	mux.HandleFunc(apiPrefix+"sessions/", server.handleSessionDetail)
	mux.HandleFunc(apiPrefix+"backends", server.handleListBackends)
}

func (server *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		sessions, err := tmux.ListSessions()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list sessions: "+err.Error())
			return
		}
		writeAPISuccess(w, sessions)

	case http.MethodPost:
		var body struct {
			Name    string   `json:"name"`
			Command []string `json:"command"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.Name != "" {
			if err := validate.SessionName(body.Name); err != nil {
				writeAPIError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		name, err := tmux.CreateSession(body.Name, body.Command...)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to create session: "+err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"name": name})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract session name from path: /api/sessions/{name}
	path := strings.TrimPrefix(r.URL.Path, server.options.Path+"api/sessions/")
	path = strings.TrimPrefix(path, "/api/sessions/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		writeAPIError(w, http.StatusBadRequest, "session name required")
		return
	}
	if err := validate.SessionName(path); err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch r.Method {
	case http.MethodGet:
		detail, err := tmux.GetSessionDetail(path)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "session not found: "+err.Error())
			return
		}
		writeAPISuccess(w, detail)

	case http.MethodDelete:
		if err := tmux.KillSession(path); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to kill session: "+err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "killed"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleListBackends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	backends := []map[string]interface{}{
		{"name": "local", "available": true},
		{"name": "tmux", "available": tmux.IsServerRunning()},
		{"name": "zellij", "available": false}, // TODO: check zellij availability
	}
	writeAPISuccess(w, backends)
}

func writeAPISuccess(w http.ResponseWriter, data interface{}) {
	resp := apiResponse{Success: true, Data: data}
	json.NewEncoder(w).Encode(resp)
}

func writeAPIError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	resp := apiResponse{Success: false, Error: message}
	json.NewEncoder(w).Encode(resp)
}
