package server

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNormalizePath_Default covers the default path normalization.
func TestNormalizePath_Default(t *testing.T) {
	srv := &Server{options: &Options{Path: "/"}}
	got := srv.normalizePath()
	if got != "/" {
		t.Errorf("expected '/', got %q", got)
	}
}

// TestNormalizePath_NoTrailingSlash covers path without trailing slash.
func TestNormalizePath_NoTrailingSlash(t *testing.T) {
	srv := &Server{options: &Options{Path: "/app"}}
	got := srv.normalizePath()
	if got != "/app/" {
		t.Errorf("expected '/app/', got %q", got)
	}
}

// TestNormalizePath_NoLeadingSlash covers path without leading slash.
func TestNormalizePath_NoLeadingSlash(t *testing.T) {
	srv := &Server{options: &Options{Path: "app/"}}
	got := srv.normalizePath()
	if got != "/app/" {
		t.Errorf("expected '/app/', got %q", got)
	}
}

// TestNormalizePath_RandomURL covers the EnableRandomURL path.
func TestNormalizePath_RandomURL(t *testing.T) {
	srv := &Server{options: &Options{EnableRandomURL: true, RandomURLLength: 16}}
	got := srv.normalizePath()
	if len(got) < 18 { // "/" + 16 chars + "/"
		t.Errorf("expected random URL path of length >= 18, got %d: %q", len(got), got)
	}
	if got[0] != '/' || got[len(got)-1] != '/' {
		t.Errorf("expected path to start and end with '/', got %q", got)
	}
}

// TestGetAppVersion_CoversBuildInfo covers the getAppVersion function.
func TestGetAppVersion_CoversBuildInfo(t *testing.T) {
	info := getAppVersion()
	if info.Version != "dev" {
		t.Errorf("expected version 'dev', got %q", info.Version)
	}
	if info.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
	// Commit may or may not be present in test builds.
}

// TestLogStartupInfo covers the logStartupInfo function (just verifies no panic).
func TestLogStartupInfo_Combos(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{"default", Options{}},
		{"permit_write", Options{PermitWrite: true}},
		{"once", Options{Once: true}},
		{"random_port", Options{Port: "0"}},
		{"all", Options{PermitWrite: true, Once: true, Port: "0"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{options: &tc.opts}
			srv.logStartupInfo() // Just verify no panic.
		})
	}
}

// TestLogListenURLs_Combos covers the logListenURLs function.
func TestLogListenURLs_Combos(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{"http_default", Options{Address: "0.0.0.0"}},
		{"https", Options{EnableTLS: true}},
		{"localhost", Options{Address: "127.0.0.1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Server{options: &tc.opts}

			// Create a listener on a random port.
			listener, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			defer listener.Close() //nolint:errcheck // test cleanup

			srv.logListenURLs(listener, "/") // Just verify no panic.
		})
	}
}

// TestHandleAICommitMessage_EmptyDiff covers empty diff field.
func TestHandleAICommitMessage_EmptyDiff(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/worktree/projects/p1/worktrees/wt1/ai-commit-message",
		bytes.NewReader([]byte(`{"diff":""}`)),
	)
	handleAICommitMessage(rec, req, "p1", "wt1")

	compareGolden(t, rec.Body.Bytes())
}

// TestHandleAICommitMessage_MissingWorktree covers missing worktree field.
func TestHandleAICommitMessage_MissingWorktree(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/worktree/projects/p1/worktrees/wt1/ai-commit-message",
		bytes.NewReader([]byte(`{"diff":"some diff content here"}`)),
	)
	handleAICommitMessage(rec, req, "p1", "wt1")

	compareGolden(t, rec.Body.Bytes())
}

// TestHandleAICommitMessage_ValidBodyNoService covers valid body but no project.
func TestHandleAICommitMessage_ValidBodyNoService(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/worktree/projects/p1/worktrees/wt1/ai-commit-message",
		bytes.NewReader([]byte(`{"diff":"-old\n+new","worktree_path":"/some/path"}`)),
	)
	handleAICommitMessage(rec, req, "p1", "wt1")

	compareGolden(t, rec.Body.Bytes())
}

// TestHandleUpload_InvalidForm covers upload with invalid multipart form.
func TestHandleUpload_InvalidForm(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost,
		"/api/upload",
		bytes.NewReader([]byte("not multipart")),
	)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=")
	srv.handleUpload(rec, req)

	compareGolden(t, rec.Body.Bytes())
}
