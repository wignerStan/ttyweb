package server

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// handleTaskStats handles GET /api/tasks/stats.
// It returns aggregate task statistics including total counts, daily completions,
// and a status breakdown.
func (server *Server) handleTaskStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if server.statsService == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "stats service not available")
		return
	}

	// Parse optional "days" query param (default 7).
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := server.statsService.GetTaskStats()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to get task stats")
		return
	}

	daily, err := server.statsService.GetDailyCompletions(days)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to get daily completions")
		return
	}

	breakdown, err := server.statsService.GetStatusBreakdown()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to get status breakdown")
		return
	}

	type response struct {
		Stats            any `json:"stats"`
		DailyCompletions any `json:"daily_completions"`
		StatusBreakdown  any `json:"status_breakdown"`
	}

	data, err := json.Marshal(response{
		Stats:            stats,
		DailyCompletions: daily,
		StatusBreakdown:  breakdown,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to marshal response")
		return
	}

	writeAPISuccessRaw(w, data)
}
