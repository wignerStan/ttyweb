package server

import (
	"net/http"
	"strconv"
	"strings"

	"ttyweb/ai"
	"ttyweb/service"
)

// aiSessionService is the global AI session service instance.
var aiSessionService = service.NewAISessionService()

// handleAISessions dispatches requests under /api/ai/sessions/...
// Routes:
//
//	GET  /api/ai/sessions             — list sessions (query param: project)
//	GET  /api/ai/sessions/:id         — session detail
//	GET  /api/ai/sessions/:id/conversation — full conversation
//	GET  /api/ai/sessions/:id/refresh — re-parse and return conversation
//	POST /api/ai/sessions/cleanup     — remove stale sessions
func (server *Server) handleAISessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Normalize the path: strip path prefix, leading and trailing slashes.
	rel := strings.TrimPrefix(r.URL.Path, server.options.Path+"api/ai/sessions")
	rel = strings.TrimPrefix(rel, "/api/ai/sessions")
	rel = strings.Trim(rel, "/")

	switch {
	case rel == "" || rel == "?"+r.URL.RawQuery:
		// GET /api/ai/sessions
		server.handleAIListSessions(w, r)
	case rel == "cleanup":
		// POST /api/ai/sessions/cleanup
		server.handleAICleanupSessions(w, r)
	default:
		// /api/ai/sessions/:id[/...]
		server.handleAISessionSubroute(w, r, rel)
	}
}

// handleAIListSessions handles GET /api/ai/sessions.
// Query param "project" filters by project path.
func (server *Server) handleAIListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")

	// Perform a fresh scan and upsert results.
	if err := refreshSessionsFromDisk(project); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to scan sessions: "+err.Error())
		return
	}

	sessions := aiSessionService.GetSessions(project)

	type sessionResponse struct {
		ID                    int     `json:"id"`
		SessionID             string  `json:"sessionId"`
		Type                  string  `json:"type"`
		ProjectPath           string  `json:"projectPath,omitempty"`
		FilePath              string  `json:"filePath"`
		Model                 string  `json:"model,omitempty"`
		Title                 string  `json:"title,omitempty"`
		SessionStartedAt      string  `json:"sessionStartedAt"`
		LastMessageAt         *string `json:"lastMessageAt,omitempty"`
		MessageCount          int     `json:"messageCount"`
		AssistantMessageCount int     `json:"assistantMessageCount"`
		FileModTime           string  `json:"fileModTime"`
		FileSize              int64   `json:"fileSize"`
	}

	data := make([]sessionResponse, 0, len(sessions))
	for _, s := range sessions {
		var lastMsg *string
		if s.LastMessageAt != nil {
			formatted := s.LastMessageAt.Format("2006-01-02T15:04:05Z07:00")
			lastMsg = &formatted
		}

		data = append(data, sessionResponse{
			ID:                    s.ID,
			SessionID:             s.SessionID,
			Type:                  s.Type,
			ProjectPath:           s.ProjectPath,
			FilePath:              s.FilePath,
			Model:                 s.Model,
			Title:                 s.Title,
			SessionStartedAt:      s.SessionStartedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastMessageAt:         lastMsg,
			MessageCount:          s.MessageCount,
			AssistantMessageCount: s.AssistantMessageCount,
			FileModTime:           s.FileModTime.Format("2006-01-02T15:04:05Z07:00"),
			FileSize:              s.FileSize,
		})
	}

	writeAPISuccess(w, data)
}

// handleAICleanupSessions handles POST /api/ai/sessions/cleanup.
func (server *Server) handleAICleanupSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	removed := aiSessionService.CleanupStaleSessions()

	writeAPISuccess(w, map[string]interface{}{
		"removed": removed,
	})
}

// handleAISessionSubroute dispatches to detail/conversation/refresh handlers.
func (server *Server) handleAISessionSubroute(w http.ResponseWriter, r *http.Request, rel string) {
	parts := strings.SplitN(rel, "/", 2)
	idStr := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid session ID: "+idStr)
		return
	}

	switch sub {
	case "":
		server.aiSessionDetail(w, r, id)
	case "conversation":
		server.aiSessionConversation(w, r, id)
	case "refresh":
		server.aiSessionRefresh(w, r, id)
	default:
		writeAPIError(w, http.StatusNotFound, "unknown route")
	}
}

// aiSessionDetail handles GET /api/ai/sessions/:id.
func (server *Server) aiSessionDetail(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	record, ok := aiSessionService.GetSession(id)
	if !ok {
		writeAPIError(w, http.StatusNotFound, "session not found")
		return
	}

	var lastMsg *string
	if record.LastMessageAt != nil {
		formatted := record.LastMessageAt.Format("2006-01-02T15:04:05Z07:00")
		lastMsg = &formatted
	}

	writeAPISuccess(w, map[string]interface{}{
		"id":                    record.ID,
		"sessionId":             record.SessionID,
		"type":                  record.Type,
		"projectPath":           record.ProjectPath,
		"filePath":              record.FilePath,
		"model":                 record.Model,
		"title":                 record.Title,
		"sessionStartedAt":      record.SessionStartedAt.Format("2006-01-02T15:04:05Z07:00"),
		"lastMessageAt":         lastMsg,
		"messageCount":          record.MessageCount,
		"assistantMessageCount": record.AssistantMessageCount,
		"fileModTime":           record.FileModTime.Format("2006-01-02T15:04:05Z07:00"),
		"fileSize":              record.FileSize,
	})
}

// aiSessionConversation handles GET /api/ai/sessions/:id/conversation.
func (server *Server) aiSessionConversation(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	messages, err := aiSessionService.GetConversation(id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to parse conversation: "+err.Error())
		return
	}
	if messages == nil {
		writeAPIError(w, http.StatusNotFound, "session not found")
		return
	}

	writeAPISuccess(w, messages)
}

// aiSessionRefresh handles GET /api/ai/sessions/:id/refresh.
func (server *Server) aiSessionRefresh(w http.ResponseWriter, r *http.Request, id int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	messages, err := aiSessionService.RefreshSession(id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to refresh conversation: "+err.Error())
		return
	}
	if messages == nil {
		writeAPIError(w, http.StatusNotFound, "session not found")
		return
	}

	writeAPISuccess(w, messages)
}

// refreshSessionsFromDisk scans Claude and Codex session directories
// and upserts results into the store.
func refreshSessionsFromDisk(projectPath string) error {
	// Scan Claude sessions.
	if projectPath != "" {
		claudeSessions, err := ai.ScanClaudeSessions(projectPath)
		if err != nil {
			return err
		}
		for _, s := range claudeSessions {
			aiSessionService.Store().Upsert(s)
		}
	} else {
		projects, err := ai.ScanClaudeProjects()
		if err != nil {
			return err
		}
		for _, project := range projects {
			sessions, scanErr := ai.ScanClaudeSessions(project)
			if scanErr != nil {
				continue
			}
			for _, s := range sessions {
				aiSessionService.Store().Upsert(s)
			}
		}
	}

	// Scan Codex sessions.
	codexSessions, err := ai.ScanCodexSessions()
	if err != nil {
		return err
	}
	for _, s := range codexSessions {
		aiSessionService.Store().Upsert(s)
	}

	return nil
}
