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
// The path is resolved to an absolute path before storage.
func registerFSRoot(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	fsRootsMu.Lock()
	defer fsRootsMu.Unlock()
	fsRoots = append(fsRoots, abs)
}

// initFSDefaults registers the current working directory as the default root.
func initFSDefaults() {
	if wd, err := os.Getwd(); err == nil {
		registerFSRoot(wd)
	}
}

// isPathAllowed returns true if the given path is under one of the registered roots.
func isPathAllowed(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	// Clean the path to resolve any ".." components.
	abs = filepath.Clean(abs)

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
	w.Header().Set("Content-Type", "application/json")

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

	entries, err := os.ReadDir(absPath)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "directory not found")
		return
	}

	var result []FSEntry
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil {
			continue
		}
		result = append(result, FSEntry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
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

	writeAPISuccess(w, result)
}
