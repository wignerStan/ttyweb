package server

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FSEntry represents a single file or directory in a listing.
type FSEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

// fsRoots holds the allowed root directories that the file browser may access.
var (
	fsRoots   = []string{}
	fsRootsMu sync.RWMutex
)

// registerFSRoot adds a directory path to the allowlist.
// The path is resolved to an absolute path and symlinks are evaluated
// before storage so that prefix comparisons in isPathAllowed are accurate.
func registerFSRoot(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	abs = filepath.Clean(abs)
	// Evaluate symlinks so that the stored root is the real path.
	// If the path does not exist yet, EvalSymlinks will error and we
	// fall back to the cleaned absolute path.
	if evaled, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		abs = evaled
	}
	fsRootsMu.Lock()
	defer fsRootsMu.Unlock()
	for _, existing := range fsRoots {
		if existing == abs {
			return
		}
	}
	fsRoots = append(fsRoots, abs)
}

// initFSDefaults registers the current working directory as the default root.
func initFSDefaults() {
	if wd, err := os.Getwd(); err == nil {
		registerFSRoot(wd)
	}
}

// isPathAllowed returns true if the given path is under one of the registered roots.
// Both the input path and registered roots have their symlinks resolved before
// comparison to prevent symlink-based directory escape.
func isPathAllowed(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	// Clean the path to resolve any ".." components.
	abs = filepath.Clean(abs)
	// Evaluate symlinks so that a symlink inside the root that points
	// outside is resolved before the prefix check. If the path does not
	// exist (e.g. has a trailing nonexistent component), EvalSymlinks
	// will error and we fall back to the cleaned absolute path.
	if evaled, evalErr := filepath.EvalSymlinks(abs); evalErr == nil {
		abs = evaled
	}

	fsRootsMu.RLock()
	defer fsRootsMu.RUnlock()
	for _, root := range fsRoots {
		if abs == root || strings.HasPrefix(abs, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// handleFSList handles GET /api/fs?path=<dir>.
// It returns a JSON array of FSEntry objects for the requested directory.
func (*Server) handleFSList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		rawPath = "."
	}

	if !isPathAllowed(rawPath) {
		writeAPIError(w, http.StatusForbidden, "path not allowed")
		return
	}

	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid path")
		return
	}

	// Defense-in-depth: resolve symlinks on absPath and re-verify it is
	// still under an allowed root. This catches any edge case where the
	// earlier isPathAllowed check passed on an unresolved path.
	if evaled, evalErr := filepath.EvalSymlinks(absPath); evalErr == nil {
		absPath = evaled
	}
	if !isPathAllowed(absPath) {
		writeAPIError(w, http.StatusForbidden, "path not allowed")
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "directory not found")
		return
	}

	writeAPISuccess(w, buildFSEntries(entries))
}

// buildFSEntries converts os.DirEntry slices into sorted FSEntry slices,
// hiding dot-files and reporting 0 size for directories.
func buildFSEntries(entries []os.DirEntry) []FSEntry {
	var result []FSEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		size := info.Size()
		if e.IsDir() {
			size = 0 // directory size is filesystem-specific; report 0 for portability
		}
		result = append(result, FSEntry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    size,
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// Sort: directories first, then alphabetically by name.
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	if result == nil {
		result = []FSEntry{}
	}
	return result
}
