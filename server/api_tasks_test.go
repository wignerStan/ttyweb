package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleTasks_GET_Empty(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks", nil)
	srv.handleTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error")
	}
	if resp.Data["total"] != float64(0) {
		t.Fatalf("expected total 0, got %v", resp.Data["total"])
	}
}

func TestHandleTasks_GET_WithPagination(t *testing.T) {
	srv := newTestServer()
	for i := 0; i < 5; i++ {
		store.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "msg"})
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks?page=1&limit=2", nil)
	srv.handleTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data["total"] != float64(5) {
		t.Fatalf("expected total 5, got %v", resp.Data["total"])
	}
	tasks := resp.Data["tasks"].([]interface{})
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestHandleTasks_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks", nil)
	srv.handleTasks(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_PostEvent(t *testing.T) {
	srv := newTestServer()
	body := `{"task_id":"t1","pane_key":"p1","event":"user_message","data":{"text":"hello"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/events", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Success bool      `json:"success"`
		Data    TaskEvent `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Data.PaneKey != "p1" {
		t.Fatalf("expected pane_key p1, got %q", resp.Data.PaneKey)
	}
	if resp.Data.Event != "user_message" {
		t.Fatalf("expected event user_message, got %q", resp.Data.Event)
	}
}

func TestHandleTaskDetail_PostEvent_MissingPaneKey(t *testing.T) {
	srv := newTestServer()
	body := `{"event":"msg"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/events", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_PostEvent_MissingEvent(t *testing.T) {
	srv := newTestServer()
	body := `{"pane_key":"p1"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/events", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_PostEvent_InvalidBody(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/events", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_GetEventsByPane(t *testing.T) {
	srv := newTestServer()
	// Use a unique pane key to avoid interference from other tests.
	store.AddTaskEvent(TaskEvent{PaneKey: "unique-pane-test", Event: "e1"})
	store.AddTaskEvent(TaskEvent{PaneKey: "other-pane", Event: "e2"})
	store.AddTaskEvent(TaskEvent{PaneKey: "unique-pane-test", Event: "e3"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/events/unique-pane-test", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Success bool        `json:"success"`
		Data    []TaskEvent `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 events for unique-pane-test, got %d", len(resp.Data))
	}
}

func TestHandleTaskDetail_GetEventsByPane_TrailingSlash(t *testing.T) {
	// Trailing slash is stripped, so /api/tasks/events/ matches "events" route
	// which is POST-only, so GET returns 405.
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/events/", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_GetEvents_MethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/events", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_CompleteTask(t *testing.T) {
	srv := newTestServer()
	te := store.AddTaskEvent(TaskEvent{PaneKey: "complete-test-pane", Event: "msg"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("/api/tasks/%d/complete", te.ID), nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	events := store.GetTaskEventsByPane("complete-test-pane")
	if len(events) == 0 {
		t.Fatal("expected to find task events")
	}
	if !events[0].Completed {
		t.Fatal("expected task to be completed")
	}
}

func TestHandleTaskDetail_CompleteTask_NotFound(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/999/complete", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_CompleteTask_InvalidID(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/abc/complete", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_UnknownEndpoint(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/unknown", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_EventsMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/tasks/events", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_EventsByPaneMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/events/p1", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleTaskDetail_CompleteMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/tasks/1/complete", nil)
	srv.handleTaskDetail(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
