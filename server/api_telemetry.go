package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleTelemetry accepts telemetry events and logs them in debug mode.
func (server *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Log telemetry events at debug level.
	if server.options.Quiet {
		// In quiet mode, do not log telemetry.
	} else {
		for _, event := range body.Events {
			log.Printf("[telemetry] %v", event)
		}
	}

	writeAPISuccess(w, map[string]string{"status": "received"})
}
