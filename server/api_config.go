package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// handleConfig reads the opencode configuration from the cwd query parameter.
// If cwd is empty or no config file is found, returns null.
func (*Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeAPISuccess(w, nil)
		return
	}

	for _, name := range []string{"opencode.json", ".opencode.json"} {
		path := filepath.Join(cwd, name)
		data, err := os.ReadFile(path) //nolint:gosec // reason: path is constructed from validated cwd query parameter
		if err != nil {
			continue
		}
		var result any
		if json.Unmarshal(data, &result) == nil {
			writeAPISuccess(w, result)
			return
		}
	}

	writeAPISuccess(w, nil)
}
