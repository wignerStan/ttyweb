package server

import (
	"encoding/json"
	"net/http"
)

// handleAICommand handles POST requests to send an AI command.
// This is a stub that returns an empty response since there is no LLM backend.
func (server *Server) handleAICommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Command string `json:"command"`
		RoleID  int    `json:"role_id"`
		PaneKey string `json:"pane_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Command == "" {
		writeAPIError(w, http.StatusBadRequest, "command is required")
		return
	}

	// No LLM backend is available. Return an empty success response.
	writeAPISuccess(w, map[string]interface{}{
		"response": "",
		"message":  "no LLM backend configured",
	})
}
