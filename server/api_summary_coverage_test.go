package server

import (
	"testing"
)

// TestExtractSummarySegmentID tests the segment ID extraction.
func TestExtractSummarySegmentID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pathPrefix string
		want       string
	}{
		{"summarize suffix", "/api/segments/seg-123/summarize", "/", "seg-123"},
		{"summary suffix", "/api/segments/seg-456/summary", "/", "seg-456"},
		{"trailing slash", "/api/segments/seg-789/", "/", "seg-789"},
		{"custom prefix", "/api/segments/seg-abc/summary", "/custom/", "/api/segments/seg-abc"},
		{"empty after prefix", "/api/segments/", "/", ""},
		{"no prefix match", "/api/other/seg-123/summary", "/", "/api/other/seg-123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSummarySegmentID(tc.path, tc.pathPrefix)
			if got != tc.want {
				t.Errorf("extractSummarySegmentID(%q, %q) = %q, want %q", tc.path, tc.pathPrefix, got, tc.want)
			}
		})
	}
}
