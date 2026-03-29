package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"ttyweb/backend"
	"ttyweb/db"
	"ttyweb/pkg/validate"
	"ttyweb/service"
)

// apiResponse is a standard envelope for API responses.
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// sessionManager returns the SessionManager for the active backend,
// or a NoSessionManager if the backend doesn't support it.
func (server *Server) sessionManager() backend.SessionManager {
	if sm, ok := server.factory.(backend.SessionManager); ok {
		return sm
	}
	return backend.NoSessionManager{}
}

// setupAPIHandlers registers REST API routes on the given mux.
func (server *Server) setupAPIHandlers(mux *http.ServeMux, pathPrefix string) {
	apiPrefix := pathPrefix + "api/"
	mux.HandleFunc(apiPrefix+"sessions", server.handleListSessions)
	mux.HandleFunc(apiPrefix+"sessions/", server.handleSessionDetail)
	mux.HandleFunc(apiPrefix+"backends", server.handleListBackends)

	// Auth
	mux.HandleFunc(apiPrefix+"auth/check", server.handleAuthCheck)
	// Profiles
	mux.HandleFunc(apiPrefix+"profiles", server.handleProfiles)
	mux.HandleFunc(apiPrefix+"profiles/", server.handleProfileDetail)
	// Groups
	mux.HandleFunc(apiPrefix+"groups", server.handleGroups)
	mux.HandleFunc(apiPrefix+"groups/", server.handleGroupDetail)
	// Snippets
	mux.HandleFunc(apiPrefix+"snippets", server.handleSnippets)
	mux.HandleFunc(apiPrefix+"snippets/", server.handleSnippetDetail)
	// Roles
	mux.HandleFunc(apiPrefix+"roles/defaults", server.handleDefaultRoles)
	mux.HandleFunc(apiPrefix+"roles", server.handleRoles)
	mux.HandleFunc(apiPrefix+"roles/", server.handleRoleDetail)
	// Tasks
	mux.HandleFunc(apiPrefix+"tasks", server.handleTasks)
	mux.HandleFunc(apiPrefix+"tasks/", server.handleTaskDetail)
	// Panes
	mux.HandleFunc(apiPrefix+"panes/status", server.handlePaneStatus)
	// AI
	mux.HandleFunc(apiPrefix+"ai/command", server.handleAICommand)
	// AI Sessions (dispatcher for list/detail/conversation/refresh/cleanup)
	mux.HandleFunc(apiPrefix+"ai/sessions", server.handleAISessions)
	mux.HandleFunc(apiPrefix+"ai/sessions/", server.handleAISessions)
	// Upload
	mux.HandleFunc(apiPrefix+"upload", server.handleUpload)
	// Config
	mux.HandleFunc(apiPrefix+"opencode-config", server.handleConfig)
	// Tmux extensions
	mux.HandleFunc(apiPrefix+"tmux/config", server.handleTmuxConfig)
	mux.HandleFunc(apiPrefix+"tmux/quick-dirs", server.handleQuickDirs)
	mux.HandleFunc(apiPrefix+"tmux/new-window", server.handleTmuxNewWindow)
	mux.HandleFunc(apiPrefix+"tmux/new-session", server.handleTmuxNewSession)
	mux.HandleFunc(apiPrefix+"tmux/tree", server.handleTmuxTree)
	mux.HandleFunc(apiPrefix+"tmux/send-keys", server.handleTmuxSendKeys)
	mux.HandleFunc(apiPrefix+"tmux/pane-mode", server.handleTmuxPaneMode)
	// Telemetry
	mux.HandleFunc(apiPrefix+"telemetry", server.handleTelemetry)
	// Log
	mux.HandleFunc(apiPrefix+"log", server.handleLog)
	// Butler
	mux.HandleFunc(apiPrefix+"butler/", server.handleButlerProxy)
	// Projects
	mux.HandleFunc(apiPrefix+"projects", server.handleProjects)
	mux.HandleFunc(apiPrefix+"projects/", server.handleProjectDetail)
	// Notepad
	mux.HandleFunc(apiPrefix+"notepad/reorder", server.handleNotepadReorder)
	mux.HandleFunc(apiPrefix+"notepad", server.handleNotepad)
	mux.HandleFunc(apiPrefix+"notepad/", server.handleNotepadDetail)
	// Task Segments
	mux.HandleFunc(apiPrefix+"segments", server.handleSegments)
	mux.HandleFunc(apiPrefix+"segments/", server.handleSegmentDetail)
	// Worktree
	setupWorktreeRoutes(mux, apiPrefix)
	// Kanban task-AI session linking
	mux.HandleFunc(apiPrefix+"kanban/tasks/", server.handleTaskAISessionLinks)
}

func (server *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		sm := server.sessionManager()
		if !sm.IsAvailable() {
			writeAPISuccess(w, []interface{}{})
			return
		}
		data, err := sm.ListSessions()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list sessions: "+err.Error())
			return
		}
		writeAPISuccessRaw(w, data)

	case http.MethodPost:
		sm := server.sessionManager()
		if !sm.IsAvailable() {
			writeAPIError(w, http.StatusServiceUnavailable, "session management not available for this backend")
			return
		}
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
		name, err := sm.CreateSession(body.Name, body.Command...)
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

	sm := server.sessionManager()
	if !sm.IsAvailable() {
		writeAPIError(w, http.StatusServiceUnavailable, "session management not available for this backend")
		return
	}

	switch r.Method {
	case http.MethodGet:
		data, err := sm.GetSessionDetail(path)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "session not found: "+err.Error())
			return
		}
		writeAPISuccessRaw(w, data)

	case http.MethodDelete:
		if err := sm.KillSession(path); err != nil {
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

	sm := server.sessionManager()
	current := server.factory.Name()

	backends := []map[string]interface{}{
		{"name": "local", "available": true, "active": current == "local command"},
		{"name": "tmux", "available": sm.IsAvailable() && current == "tmux", "active": current == "tmux"},
		{"name": "zellij", "available": sm.IsAvailable() && current == "zellij", "active": current == "zellij"},
	}

	// Check availability for backends that aren't the current one.
	// The local backend never has session management.
	// For non-active backends, show them as available if the binary exists,
	// but session APIs only work with the active backend.
	if current != "tmux" {
		for _, b := range backends {
			if b["name"] == "tmux" {
				b["available"] = false // can't manage sessions from a different active backend
			}
		}
	}
	if current != "zellij" {
		for _, b := range backends {
			if b["name"] == "zellij" {
				b["available"] = false
			}
		}
	}

	writeAPISuccess(w, backends)
}

func writeAPISuccess(w http.ResponseWriter, data interface{}) {
	raw, err := json.Marshal(data)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to marshal response")
		return
	}
	writeAPISuccessRaw(w, raw)
}

func writeAPISuccessRaw(w http.ResponseWriter, data json.RawMessage) {
	resp := apiResponse{Success: true, Data: data}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeAPIError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	resp := apiResponse{Success: false, Error: message}
	_ = json.NewEncoder(w).Encode(resp)
}

// projectService returns a lazily-initialized ProjectService backed by SQLite.
// Returns an error if the database cannot be opened.
var (
	projectServiceOnce     sync.Once
	projectServiceInstance *service.ProjectService
	errProjectService      error
)

func projectService() (*service.ProjectService, error) {
	projectServiceOnce.Do(func() {
		gormDB, err := db.GetDB()
		if err != nil {
			errProjectService = fmt.Errorf("get project database: %w", err)
			return
		}
		projectServiceInstance = service.NewProjectService(gormDB)
	})
	if errProjectService != nil {
		return nil, errProjectService
	}
	return projectServiceInstance, nil
}
