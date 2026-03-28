package server

import (
	"net/http"
)

// handleConfig returns the opencode configuration.
// This is a stub that returns null configs since opencode is not integrated.
func (server *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeAPISuccess(w, map[string]interface{}{
		"opencode":      nil,
		"oh_my_opencode": nil,
	})
}
