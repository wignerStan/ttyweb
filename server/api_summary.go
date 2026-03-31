package server

import (
	"log"
	"net/http"
	"strings"

	"ttyweb/db"
)

// handleSummarizeSegment handles POST /api/segments/{id}/summarize.
// It generates an AI-powered summary for the given task segment.
func (server *Server) handleSummarizeSegment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if server.summaryService == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "summary service not available")
		return
	}

	// Extract segment ID from URL path: /api/segments/{id}/summarize
	segmentID := extractSummarySegmentID(r.URL.Path, server.options.Path)
	if segmentID == "" {
		writeAPIError(w, http.StatusBadRequest, "segment ID required")
		return
	}

	summary, err := server.summaryService.GenerateSummary(r.Context(), segmentID)
	if err != nil {
		log.Printf("failed to generate summary: %v", err)
		if strings.Contains(err.Error(), "segment not found") {
			writeAPIError(w, http.StatusNotFound, "segment not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "failed to generate summary")
		return
	}

	writeAPISuccess(w, summary)
}

// handleGetSummary handles GET /api/segments/{id}/summary.
// It returns the existing summary for a segment, or 404 if none exists.
func (server *Server) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	if server.summaryService == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "summary service not available")
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	segmentID := extractSummarySegmentID(r.URL.Path, server.options.Path)
	if segmentID == "" {
		writeAPIError(w, http.StatusBadRequest, "segment ID required")
		return
	}

	summary, err := server.summaryService.GetSummary(segmentID)
	if err != nil {
		log.Printf("failed to get summary: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to get summary")
		return
	}
	if summary == nil {
		writeAPIError(w, http.StatusNotFound, "summary not found")
		return
	}

	writeAPISuccess(w, summary)
}

// handleListSummaries handles GET /api/segments/summaries.
// It returns all summaries ordered by most recent, limited to 100.
func (server *Server) handleListSummaries(w http.ResponseWriter, r *http.Request) {
	if server.summaryService == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "summary service not available")
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summaries, err := server.summaryService.ListSummaries()
	if err != nil {
		log.Printf("failed to list summaries: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to list summaries")
		return
	}

	if summaries == nil {
		summaries = make([]db.TaskSummary, 0)
	}
	writeAPISuccess(w, summaries)
}

// extractSummarySegmentID extracts the segment ID from a URL path.
// Supports both /api/segments/{id}/summarize and /api/segments/{id}/summary.
func extractSummarySegmentID(path, pathPrefix string) string {
	base := strings.TrimPrefix(path, pathPrefix+"api/segments/")
	// Remove trailing /summarize or /summary suffix.
	for _, suffix := range []string{"/summarize", "/summary"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}
	base = strings.TrimSuffix(base, "/")
	return base
}
