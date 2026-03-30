package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// handleEditorOpen generates an editor URL (vscode:// or cursor://) for opening a file.
func (*Server) handleEditorOpen(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Col  int    `json:"col"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" {
		writeAPIError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid path")
		return
	}

	// Determine protocol URL based on file extension or editor.
	scheme := "vscode" // default
	if strings.Contains(absPath, ".cursor") {
		scheme = "cursor"
	}

	url := fmt.Sprintf("%s://file%s", scheme, absPath)
	if req.Line > 0 {
		url += fmt.Sprintf(":%d", req.Line)
		if req.Col > 0 {
			url += fmt.Sprintf(":%d", req.Col)
		}
	}

	writeAPISuccess(w, map[string]string{"url": url, "path": absPath})
}
