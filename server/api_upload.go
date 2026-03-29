package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const uploadDir = "/tmp/ttyweb-uploads"

// handleUpload handles POST multipart file uploads.
// Accepts a "file" field in the form, saves to /tmp/ttyweb-uploads/,
// and returns {filename, path, size, type}.
func (_ *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit upload size to 50 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeAPIError(w, http.StatusBadRequest, "failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "file field required: "+err.Error())
		return
	}
	defer func() { _ = file.Close() }()

	// Ensure upload directory exists.
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create upload directory: "+err.Error())
		return
	}

	// Generate a unique filename to avoid collisions.
	ext := filepath.Ext(header.Filename)
	timestamp := time.Now().Format("20060102-150405")
	uniqueName := fmt.Sprintf("%s-%d%s", timestamp, time.Now().UnixNano(), ext)
	destPath := filepath.Join(uploadDir, uniqueName)

	dst, err := os.Create(destPath)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to create file: "+err.Error())
		return
	}
	defer func() { _ = dst.Close() }()

	written, err := io.Copy(dst, file)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to write file: "+err.Error())
		return
	}

	// Determine content type from the uploaded file's header.
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	writeAPISuccess(w, map[string]any{
		"filename": header.Filename,
		"path":     destPath,
		"size":     written,
		"type":     strings.Split(contentType, ";")[0],
	})
}
