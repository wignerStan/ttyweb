package server

import (
	"testing"
)

// NOTE: logResponseWriter.Hijack panics when the underlying ResponseWriter
// doesn't implement http.Hijacker (nil type assertion). This is a known
// limitation — the Hijack path is only exercised when a real WebSocket
// connection triggers it, which always has a Hijack-capable ResponseWriter.

// TestExtractIP_Cases tests the extractIP helper function.
func TestExtractIP_Cases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"10.0.0.1", "10.0.0.1"},
		{"", ""},
	}
	for _, tc := range tests {
		got := extractIP(tc.input)
		if got != tc.want {
			t.Errorf("extractIP(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
