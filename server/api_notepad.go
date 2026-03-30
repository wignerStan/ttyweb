package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"ttyweb/service"
)

// handleNotepad handles GET (list) and POST (create) for notepad notes.
func (server *Server) handleNotepad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		var projectID *string
		if pid := r.URL.Query().Get("project_id"); pid != "" {
			projectID = &pid
		}
		notes, err := server.noteSvc.ListNotes(projectID)
		if err != nil {
			log.Printf("failed to list notes: %v", err)
			writeAPIError(w, http.StatusInternalServerError, "failed to list notes")
			return
		}
		writeAPISuccess(w, notes)

	case http.MethodPost:
		var body struct {
			Name      string  `json:"name"`
			Content   string  `json:"content"`
			ProjectID *string `json:"project_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		var opts []service.NoteOption
		if body.ProjectID != nil {
			opts = append(opts, service.WithProjectID(*body.ProjectID))
		}
		note, err := server.noteSvc.CreateNote(body.Name, body.Content, opts...)
		if err != nil {
			log.Printf("failed to create note: %v", err)
			writeAPIError(w, http.StatusInternalServerError, "failed to create note")
			return
		}
		writeAPISuccess(w, note)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNotepadDetail handles GET, PUT, and DELETE for a single notepad note by ID.
func (server *Server) handleNotepadDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from path: /api/notepad/{id}
	prefix := server.options.Path + "api/notepad/"
	id := strings.TrimPrefix(r.URL.Path, prefix)
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "note ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		note, err := server.noteSvc.GetNote(id)
		if err != nil {
			log.Printf("failed to get note %s: %v", id, err)
			status := http.StatusInternalServerError
			if errors.Is(err, service.ErrNotFound) {
				status = http.StatusNotFound
			}
			writeAPIError(w, status, "failed to get note")
			return
		}
		writeAPISuccess(w, note)

	case http.MethodPut:
		var body struct {
			Name    *string `json:"name,omitempty"`
			Content *string `json:"content,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		note, err := server.noteSvc.UpdateNote(id, body.Name, body.Content)
		if err != nil {
			log.Printf("failed to update note %s: %v", id, err)
			if errors.Is(err, service.ErrNotFound) {
				writeAPIError(w, http.StatusNotFound, "note not found")
				return
			}
			writeAPIError(w, http.StatusBadRequest, "failed to update note")
			return
		}
		writeAPISuccess(w, note)

	case http.MethodDelete:
		if err := server.noteSvc.DeleteNote(id); err != nil {
			log.Printf("failed to delete note %s: %v", id, err)
			status := http.StatusInternalServerError
			if errors.Is(err, service.ErrNotFound) {
				status = http.StatusNotFound
			}
			writeAPIError(w, status, "failed to delete note")
			return
		}
		writeAPISuccess(w, map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNotepadReorder handles PATCH for reordering notepad notes.
func (server *Server) handleNotepadReorder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reorders []service.NoteReorder
	if err := json.NewDecoder(r.Body).Decode(&reorders); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := server.noteSvc.ReorderNotes(reorders); err != nil {
		log.Printf("failed to reorder notes: %v", err)
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrNotFound) {
			status = http.StatusNotFound
		}
		writeAPIError(w, status, "failed to reorder notes")
		return
	}

	writeAPISuccess(w, map[string]string{"status": "reordered"})
}
