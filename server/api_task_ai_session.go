package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"ttyweb/service"
)

// taskAISessionSvc is the global service for task-AI session linking.
var taskAISessionSvc = service.NewTaskAISessionService()

// handleTaskAISessionLinks handles sub-routes under /api/kanban/tasks/{id}/sessions:
//
//	POST   /api/kanban/tasks/{id}/sessions       — link an AI session
//	GET    /api/kanban/tasks/{id}/sessions       — list linked sessions
//	DELETE /api/kanban/tasks/{id}/sessions/{sid} — unlink an AI session
func (server *Server) handleTaskAISessionLinks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	prefix := server.options.Path + "api/kanban/tasks/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")

	// relative is "{taskID}/sessions" or "{taskID}/sessions/{aiSessionID}".
	segments := strings.SplitN(relative, "/", 3)
	if len(segments) < 2 || segments[1] != "sessions" {
		writeAPIError(w, http.StatusBadRequest, "invalid path")
		return
	}

	taskID := segments[0]
	aiSessionID := ""
	if len(segments) == 3 {
		aiSessionID = segments[2]
	}

	methodNotAllowed := func() {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	// Dispatch by HTTP method.
	switch r.Method {
	case http.MethodPost:
		if aiSessionID != "" {
			methodNotAllowed()
			return
		}
		var body struct {
			AISessionID string `json:"ai_session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		link, err := taskAISessionSvc.LinkSession(taskID, body.AISessionID)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, link)

	case http.MethodGet:
		if aiSessionID != "" {
			methodNotAllowed()
			return
		}
		links, err := taskAISessionSvc.ListLinkedSessions(taskID)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPISuccess(w, links)

	case http.MethodDelete:
		if aiSessionID == "" {
			methodNotAllowed()
			return
		}
		if err := taskAISessionSvc.UnlinkSession(taskID, aiSessionID); err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, map[string]string{"status": "unlinked"})

	default:
		methodNotAllowed()
	}
}
