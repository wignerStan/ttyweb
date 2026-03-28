package server

import (
	"net/http"
)

// handleButlerProxy is a stub handler for butler proxy requests.
// All GET and POST requests under /api/butler/* return an empty success response.
func (server *Server) handleButlerProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet, http.MethodPost:
		writeAPISuccess(w, map[string]interface{}{
			"success": true,
			"data":    nil,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
