package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// handleTasks handles GET (list tasks with pagination).
func (*Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	tasks, total := store.ListTasks(page, limit)
	writeAPISuccess(w, map[string]any{
		"tasks": tasks,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// handleTaskDetail handles sub-routes under /api/tasks/:
//   - GET  /api/tasks/events/{paneKey}  — list conversation events for a pane
//   - POST /api/tasks/events            — create a conversation event
//   - POST /api/tasks/{id}/complete     — mark a task complete
func (server *Server) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	prefix := server.options.Path + "api/tasks/"
	relative := strings.TrimPrefix(r.URL.Path, prefix)
	relative = strings.TrimSuffix(relative, "/")

	// Route: /api/tasks/events
	if relative == "events" {
		switch r.Method {
		case http.MethodPost:
			var body struct {
				TaskID  string         `json:"task_id"`
				PaneKey string         `json:"pane_key"`
				Event   string         `json:"event"`
				Data    map[string]any `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeAPIError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if body.PaneKey == "" {
				writeAPIError(w, http.StatusBadRequest, "pane_key is required")
				return
			}
			if body.Event == "" {
				writeAPIError(w, http.StatusBadRequest, "event is required")
				return
			}
			created := store.AddTaskEvent(&TaskEvent{
				TaskID:  body.TaskID,
				PaneKey: body.PaneKey,
				Event:   body.Event,
				Data:    body.Data,
			})
			writeAPISuccess(w, created)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Route: /api/tasks/events/{paneKey}
	if strings.HasPrefix(relative, "events/") {
		paneKey := strings.TrimPrefix(relative, "events/")
		if paneKey == "" {
			writeAPIError(w, http.StatusBadRequest, "pane key required")
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		events := store.GetTaskEventsByPane(paneKey)
		writeAPISuccess(w, events)
		return
	}

	// Route: /api/tasks/{id}/complete
	if strings.HasSuffix(relative, "/complete") {
		idStr := strings.TrimSuffix(relative, "/complete")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid task ID")
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := store.CompleteTask(id); err != nil {
			log.Printf("failed to complete task: %v", err)
			writeAPIError(w, http.StatusNotFound, "task not found")
			return
		}
		writeAPISuccess(w, map[string]string{"status": "completed"})
		return
	}

	writeAPIError(w, http.StatusBadRequest, "unknown task endpoint")
}
