package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleSegments handles GET (list) and POST (create) for task segments.
func (server *Server) handleSegments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		session := r.URL.Query().Get("session")
		segments, err := server.segmentService.ListSegments(session)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list segments: "+err.Error())
			return
		}
		writeAPISuccess(w, segments)

	case http.MethodPost:
		var body struct {
			SessionName string `json:"session_name"`
			WindowName  string `json:"window_name"`
			PaneIndex   int    `json:"pane_index"`
			TaskTitle   string `json:"task_title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.TaskTitle == "" {
			writeAPIError(w, http.StatusBadRequest, "task_title is required")
			return
		}
		segment, err := server.segmentService.CreateSegment(
			body.SessionName, body.WindowName, body.PaneIndex, body.TaskTitle,
		)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to create segment: "+err.Error())
			return
		}
		writeAPISuccess(w, segment)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSegmentDetail handles sub-routes under /api/segments/:
//
//	GET  /api/segments/{id}/detail   — full detail with messages, commands, summary
//	GET  /api/segments/{id}/messages — list chat messages
//	GET  /api/segments/{id}/commands — list command records
//	PATCH /api/segments/{id}         — update segment title and/or status
//	GET  /api/segments/{id}          — returns segment detail as convenience
func (server *Server) handleSegmentDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	relative := strings.TrimPrefix(r.URL.Path, server.options.Path+"api/segments/")
	relative = strings.TrimSuffix(relative, "/")
	if relative == "" {
		writeAPIError(w, http.StatusBadRequest, "segment ID required")
		return
	}

	// Check sub-routes that share the pattern: extract ID from suffix, require GET.
	if id, suffix, ok := splitSubRoute(relative); ok {
		if server.handleSegmentSubRoute(w, r, id, suffix) {
			return
		}
	}

	// Route: PATCH /api/segments/{id}
	if r.Method == http.MethodPatch {
		var body struct {
			TaskTitle  *string `json:"task_title"`
			TaskStatus string  `json:"task_status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.TaskTitle == nil && body.TaskStatus == "" {
			writeAPIError(w, http.StatusBadRequest, "task_title or task_status is required")
			return
		}
		segment, err := server.segmentService.UpdateSegment(relative, body.TaskTitle, body.TaskStatus)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, segment)
		return
	}

	// Route: GET /api/segments/{id} — returns segment detail as convenience
	if r.Method == http.MethodGet {
		detail, err := server.segmentService.GetSegmentDetail(relative)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
			return
		}
		writeAPISuccess(w, detail)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleSegmentSubRoute handles GET sub-routes for segments (detail, messages, commands).
// Returns true if the request was handled.
func (server *Server) handleSegmentSubRoute(w http.ResponseWriter, r *http.Request, id, suffix string) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "segment ID required")
		return true
	}

	switch suffix {
	case "detail":
		detail, err := server.segmentService.GetSegmentDetail(id)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err.Error())
		} else {
			writeAPISuccess(w, detail)
		}
	case "messages":
		messages, err := server.segmentService.ListMessages(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list messages: "+err.Error())
		} else {
			writeAPISuccess(w, messages)
		}
	case "commands":
		commands, err := server.segmentService.ListCommands(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list commands: "+err.Error())
		} else {
			writeAPISuccess(w, commands)
		}
	default:
		return false
	}
	return true
}

// splitSubRoute splits a path like "abc123/detail" into ("abc123", "detail", true).
// Returns ("", "", false) if the path contains no slash.
func splitSubRoute(path string) (id string, subRoute string, ok bool) {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "", "", false
	}
	return path[:idx], path[idx+1:], true
}
