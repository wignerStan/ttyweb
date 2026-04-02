package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestIsPathAllowed_SymlinkEscape verifies that a symlink inside an allowed root
// that points outside the root is rejected. This is the core symlink escape bug:
// without filepath.EvalSymlinks, the path passes the prefix check even though
// the resolved target is outside the allowed directory.
func TestIsPathAllowed_SymlinkEscape(t *testing.T) {
	// Create a temp root and an outside directory.
	root := t.TempDir()
	outside := t.TempDir()

	// Write a marker file in the outside dir to prove we could read it.
	marker := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(marker, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	// Create a symlink inside root pointing to the outside directory.
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Register root as an allowed path.
	registerFSRoot(root)

	// The symlink path itself should be allowed (it's inside root).
	// But the *target* of the symlink through traversal should not escape.
	// Test that isPathAllowed resolves symlinks before the prefix check.
	escapedPath := filepath.Join(link, "secret.txt")

	// Without the fix, this returns true (the unresolved path is under root).
	// With the fix, it returns false (the resolved path is outside root).
	if isPathAllowed(escapedPath) {
		t.Error("isPathAllowed should reject symlink that escapes allowed root")
	}
}

// TestIsPathAllowed_NormalSubdirectory verifies that a normal (non-symlink)
// subdirectory under a registered root is accepted.
func TestIsPathAllowed_NormalSubdirectory(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	registerFSRoot(root)

	if !isPathAllowed(sub) {
		t.Error("isPathAllowed should accept a normal subdirectory under root")
	}
}

// TestIsPathAllowed_RootItself verifies that the root path itself is accepted.
func TestIsPathAllowed_RootItself(t *testing.T) {
	root := t.TempDir()
	registerFSRoot(root)

	if !isPathAllowed(root) {
		t.Error("isPathAllowed should accept the root itself")
	}
}

// TestHandleFSList_SymlinkEscape verifies the full HTTP handler rejects a
// directory listing through a symlink that escapes the allowed root.
func TestHandleFSList_SymlinkEscape(t *testing.T) {
	srv := newTestServer()

	root := t.TempDir()
	outside := t.TempDir()

	// Put something in outside so ReadDir would succeed if not blocked.
	if err := os.WriteFile(filepath.Join(outside, "leaked.txt"), []byte("no"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Symlink root/escape -> outside
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	registerFSRoot(root)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+link, nil)
	srv.handleFSList(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for symlink escape, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// TestHandleFSList_ValidSymlinkInsideRoot verifies that a symlink pointing
// to another location *within* the same root is still allowed.
func TestHandleFSList_ValidSymlinkInsideRoot(t *testing.T) {
	srv := newTestServer()

	root := t.TempDir()

	// Create a real subdirectory and a symlink pointing to it (both under root).
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	linkDir := filepath.Join(root, "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	registerFSRoot(root)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+linkDir, nil)
	srv.handleFSList(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for symlink within root, got %d; body: %s", rec.Code, rec.Body.String())
	}
}

// TestRegisterFSRoot_EvalSymlinks verifies that registerFSRoot resolves
// symlinks in the registered root path itself.
func TestRegisterFSRoot_EvalSymlinks(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "rootlink")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	registerFSRoot(linkDir)

	// The real directory should now be allowed (since the root was resolved).
	if !isPathAllowed(realDir) {
		t.Error("isPathAllowed should accept the real dir when root was registered via symlink")
	}
}
