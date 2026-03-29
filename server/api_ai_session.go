package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"ttyweb/ai"
	"ttyweb/service"
)

const isoTimeFormat = "2006-01-02T15:04:05Z07:00"

// aiSessionService is the global AI session service instance.
var aiSessionService = service.NewAISessionService()

// aiSessionResponse is the JSON shape for a single AI session in API responses.
type aiSessionResponse struct {
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

// formatOptionalTime returns a pointer to the formatted time string, or nil if t is nil.
func formatOptionalTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(isoTimeFormat)
	return &formatted
}

// newAISessionResponse converts a service record to an API response.
func newAISessionResponse(r service.AISessionRecord) aiSessionResponse {
	return aiSessionResponse{
		ID:                    r.ID,
		SessionID:             r.SessionID,
		Type:                  r.Type,
		ProjectPath:           r.ProjectPath,
		FilePath:              r.FilePath,
		Model:                 r.Model,
		Title:                 r.Title,
		SessionStartedAt:      r.SessionStartedAt.Format(isoTimeFormat),
		LastMessageAt:         formatOptionalTime(r.LastMessageAt),
		MessageCount:          r.MessageCount,
		AssistantMessageCount: r.AssistantMessageCount,
		FileModTime:           r.FileModTime.Format(isoTimeFormat),
		FileSize:              r.FileSize,
	}
}

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

	rel := strings.TrimPrefix(r.URL.Path, server.options.Path+"api/ai/sessions")
	rel = strings.TrimPrefix(rel, "/api/ai/sessions")
	rel = strings.Trim(rel, "/")

	switch rel {
	case "", "?" + r.URL.RawQuery:
		server.handleAIListSessions(w, r)
	case "cleanup":
		server.handleAICleanupSessions(w, r)
	default:
		server.handleAISessionSubroute(w, r, rel)
	}
}

// handleAIListSessions handles GET /api/ai/sessions.
func (server *Server) handleAIListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	project := r.URL.Query().Get("project")
	if err := refreshSessionsFromDisk(project); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to scan sessions: "+err.Error())
		return
	}

	sessions := aiSessionService.GetSessions(project)
	data := make([]aiSessionResponse, 0, len(sessions))
	for _, s := range sessions {
		data = append(data, newAISessionResponse(s))
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
	writeAPISuccess(w, map[string]interface{}{"removed": removed})
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
		server.aiSessionConversationOrRefresh(w, r, id, false)
	case "refresh":
		server.aiSessionConversationOrRefresh(w, r, id, true)
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

	writeAPISuccess(w, newAISessionResponse(record))
}

// aiSessionConversationOrRefresh handles GET /api/ai/sessions/:id/conversation
// and GET /api/ai/sessions/:id/refresh. When refresh is true, it re-parses
// the session file before returning the conversation.
func (server *Server) aiSessionConversationOrRefresh(w http.ResponseWriter, r *http.Request, id int, refresh bool) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var messages []ai.ConversationMessage
	var err error
	if refresh {
		messages, err = aiSessionService.RefreshSession(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to refresh conversation: "+err.Error())
			return
		}
	} else {
		messages, err = aiSessionService.GetConversation(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to parse conversation: "+err.Error())
			return
		}
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
	if projectPath != "" {
		if err := upsertClaudeSessions(projectPath); err != nil {
			return err
		}
	} else if err := upsertAllClaudeSessions(); err != nil {
		return err
	}

	codexSessions, err := ai.ScanCodexSessions()
	if err != nil {
		return errors.Wrapf(err, "refreshSessionsFromDisk: scanning codex sessions")
	}
	for _, s := range codexSessions {
		aiSessionService.Store().Upsert(s)
	}

	return nil
}

// upsertClaudeSessions scans a single project path for Claude sessions.
func upsertClaudeSessions(projectPath string) error {
	claudeSessions, err := ai.ScanClaudeSessions(projectPath)
	if err != nil {
		return errors.Wrapf(err, "upsertClaudeSessions: scanning %q", projectPath)
	}
	for _, s := range claudeSessions {
		aiSessionService.Store().Upsert(s)
	}
	return nil
}

// upsertAllClaudeSessions scans all known Claude projects for sessions.
func upsertAllClaudeSessions() error {
	projects, err := ai.ScanClaudeProjects()
	if err != nil {
		return errors.Wrapf(err, "upsertAllClaudeSessions: scanning projects")
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
	return nil
}
