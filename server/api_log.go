package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleLog accepts client-side log entries and forwards them via log.Printf.
func (_ *Server) handleLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Level   string                 `json:"level"`
		Message string                 `json:"message"`
		Data    map[string]any `json:"data"`
		UA      string                 `json:"ua"`
		URL     string                 `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Message == "" {
		writeAPIError(w, http.StatusBadRequest, "message is required")
		return
	}

	level := body.Level
	if level == "" {
		level = "info"
	}

	log.Printf("[client:%s] %s (ua=%s, url=%s)", level, body.Message, body.UA, body.URL)

	writeAPISuccess(w, map[string]string{"status": "logged"})
}
