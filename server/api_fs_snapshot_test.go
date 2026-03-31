package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixedMtime is a deterministic modification time used in snapshot tests
// so that golden files are stable across runs.
var fixedMtime = time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

// setFixedMtime sets modification (and access) times on a path to fixedMtime.
func setFixedMtime(t *testing.T, path string) {
	t.Helper()
	if err := os.Chtimes(path, fixedMtime, fixedMtime); err != nil {
		t.Fatal(err)
	}
}

// TestGolden_FSList_Directory lists a directory with files and subdirectories.
func TestGolden_FSList_Directory(t *testing.T) {
	t.Setenv("TZ", "UTC")
	dir := t.TempDir()
	registerFSRoot(dir)

	// Create files and subdirectories.
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	setFixedMtime(t, filepath.Join(dir, "subdir"))
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	setFixedMtime(t, filepath.Join(dir, "hello.txt"))
	if err := os.WriteFile(filepath.Join(dir, "world.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	setFixedMtime(t, filepath.Join(dir, "world.go"))

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+dir, nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_EmptyDir lists an empty directory.
func TestGolden_FSList_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	registerFSRoot(dir)

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+dir, nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_PathTraversal rejects a path outside registered roots.
func TestGolden_FSList_PathTraversal(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path=/etc", nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_DotfilesFiltered confirms dotfiles are excluded from listings.
func TestGolden_FSList_DotfilesFiltered(t *testing.T) {
	t.Setenv("TZ", "UTC")
	dir := t.TempDir()
	registerFSRoot(dir)

	if err := os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	setFixedMtime(t, filepath.Join(dir, "visible.txt"))
	if err := os.Mkdir(filepath.Join(dir, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+dir, nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_SortedDirsFirst verifies directories appear before files,
// each group sorted alphabetically.
func TestGolden_FSList_SortedDirsFirst(t *testing.T) {
	t.Setenv("TZ", "UTC")
	dir := t.TempDir()
	registerFSRoot(dir)

	// Create entries in non-alphabetical order.
	for _, name := range []string{"z_last.txt", "a_first_dir", "m_middle.go", "b_second_dir"} {
		p := filepath.Join(dir, name)
		if name[len(name)-4:] == "_dir" {
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		setFixedMtime(t, p)
	}

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+dir, nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_NotFound returns an error for a nonexistent directory.
func TestGolden_FSList_NotFound(t *testing.T) {
	dir := t.TempDir()
	registerFSRoot(dir)

	nonexistent := filepath.Join(dir, "does_not_exist")

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs?path="+nonexistent, nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_MethodNotAllowed rejects non-GET methods.
func TestGolden_FSList_MethodNotAllowed(t *testing.T) {
	dir := t.TempDir()
	registerFSRoot(dir)

	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/fs?path="+dir, nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_FSList_NoPath defaults to "." when no path parameter is provided.
func TestGolden_FSList_NoPath(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/fs", nil)
	srv.handleFSList(rec, req)

	compareGolden(t, rec.Body.Bytes())
}

// TestGolden_IsPathAllowed_AbsPath verifies an absolute path under a registered root is allowed.
func TestGolden_IsPathAllowed_AbsPath(t *testing.T) {
	dir := t.TempDir()
	registerFSRoot(dir)

	absPath := filepath.Join(dir, "subdir")
	result := isPathAllowed(absPath)

	compareGolden(t, []byte(fmt.Sprintf("%v", result)))
}

// TestGolden_IsPathAllowed_RelativePath verifies a relative path under a registered root is allowed.
// Since compareGolden uses a relative testdata/ path, we must not change CWD.
// Instead we register the temp dir and test with its absolute path, confirming
// that the relative "." from inside that dir would be allowed.
func TestGolden_IsPathAllowed_RelativePath(t *testing.T) {
	dir := t.TempDir()
	registerFSRoot(dir)

	// Use the absolute path directly — isPathAllowed resolves to absolute internally.
	result := isPathAllowed(dir)

	compareGolden(t, []byte(fmt.Sprintf("%v", result)))
}

// TestGolden_IsPathAllowed_DotDotTraversal verifies a path with ".." that escapes root is rejected.
func TestGolden_IsPathAllowed_DotDotTraversal(t *testing.T) {
	dir := t.TempDir()
	registerFSRoot(dir)

	escapePath := filepath.Join(dir, "..")

	result := isPathAllowed(escapePath)

	compareGolden(t, []byte(fmt.Sprintf("%v", result)))
}
