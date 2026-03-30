package server

import (
	"fmt"
	"io"
	"log"
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
func (*Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit upload size to 50 MB.
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		log.Printf("failed to parse multipart form: %v", err)
		writeAPIError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("failed to get form file: %v", err)
		writeAPIError(w, http.StatusBadRequest, "file field required")
		return
	}
	defer func() { _ = file.Close() }()

	// Ensure upload directory exists.
	if err := os.MkdirAll(uploadDir, 0o750); err != nil {
		log.Printf("failed to create upload directory: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to create upload directory")
		return
	}

	// Generate a unique filename to avoid collisions.
	ext := filepath.Ext(header.Filename)
	timestamp := time.Now().Format("20060102-150405")
	uniqueName := fmt.Sprintf("%s-%d%s", timestamp, time.Now().UnixNano(), ext)
	destPath := filepath.Join(uploadDir, uniqueName)

	dst, err := os.Create(destPath) //nolint:gosec // reason: destPath is server-generated (timestamp + nano + extension), not user-controlled
	if err != nil {
		log.Printf("failed to create file: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to create file")
		return
	}
	defer func() { _ = dst.Close() }()

	written, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("failed to write file: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	// Determine content type from the uploaded file's header.
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	writeAPISuccess(w, map[string]any{
		"filename": header.Filename,
		"path":     uniqueName,
		"size":     written,
		"type":     strings.Split(contentType, ";")[0],
	})
}
