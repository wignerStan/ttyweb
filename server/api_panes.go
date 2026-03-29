package server

import (
	"encoding/json"
	"net/http"
)

// handlePaneStatus handles GET (all pane statuses) and PUT (update pane status).
func (_ *Server) handlePaneStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		statuses := store.GetPaneStatuses()
		writeAPISuccess(w, statuses)

	case http.MethodPut:
		var body struct {
			PaneKey string `json:"pane_key"`
			Status  string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.PaneKey == "" {
			writeAPIError(w, http.StatusBadRequest, "pane_key is required")
			return
		}
		if body.Status == "" {
			writeAPIError(w, http.StatusBadRequest, "status is required")
			return
		}
		store.SetPaneStatus(body.PaneKey, body.Status)
		writeAPISuccess(w, map[string]string{
			"pane_key": body.PaneKey,
			"status":   body.Status,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
