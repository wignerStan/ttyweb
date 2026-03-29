//go:build integration

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

// Integration tests exercise full request/response cycles through the server
// handlers, backed by real SQLite. Each test verifies a complete CRUD lifecycle
// or cross-endpoint flow using unique data to avoid interference.

func postJSON(t *testing.T, handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handler(rec, req)
	return rec
}

func getJSON(t *testing.T, handler http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	handler(rec, req)
	return rec
}

func putJSON(t *testing.T, handler http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	handler(rec, req)
	return rec
}

func deleteReq(t *testing.T, handler http.HandlerFunc, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, path, nil)
	handler(rec, req)
	return rec
}

func decodeResp(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return m
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected %d, got %d; body: %s", want, rec.Code, rec.Body.String())
	}
}

func TestIntegration_ProfileLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Create
	rec := postJSON(t, srv.handleProfiles, "/api/profiles",
		`{"profile_key":"integ-prof","name":"Original","sort_order":1}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// Verify in list
	rec = getJSON(t, srv.handleProfiles, "/api/profiles")
	assertStatus(t, rec, http.StatusOK)
	data := decodeResp(t, rec)["data"].([]any)
	found := false
	for _, item := range data {
		if int(item.(map[string]any)["id"].(float64)) == id {
			found = true
		}
	}
	if !found {
		t.Fatal("profile not in list")
	}

	// Update
	rec = putJSON(t, srv.handleProfileDetail, fmt.Sprintf("/api/profiles/%d", id),
		`{"profile_key":"integ-prof","name":"Updated","sort_order":5}`)
	assertStatus(t, rec, http.StatusOK)
	resp := decodeResp(t, rec)
	if resp["data"].(map[string]any)["name"] != "Updated" {
		t.Fatal("name not updated")
	}

	// Delete and verify gone
	rec = deleteReq(t, srv.handleProfileDetail, fmt.Sprintf("/api/profiles/%d", id))
	assertStatus(t, rec, http.StatusOK)
	for _, p := range store.ListProfiles() {
		if p.ID == id {
			t.Fatal("profile still exists")
		}
	}
}

func TestIntegration_GroupLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleGroups, "/api/groups",
		`{"group_name":"integ-grp","sort_order":1,"profile_key":"pk-integ"}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// Filter by profile_key
	rec = getJSON(t, srv.handleGroups, "/api/groups?profile_key=pk-integ")
	assertStatus(t, rec, http.StatusOK)
	items := decodeResp(t, rec)["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 group, got %d", len(items))
	}

	rec = putJSON(t, srv.handleGroupDetail, fmt.Sprintf("/api/groups/%d", id),
		`{"group_name":"renamed","sort_order":5,"profile_key":"pk-integ"}`)
	assertStatus(t, rec, http.StatusOK)

	rec = deleteReq(t, srv.handleGroupDetail, fmt.Sprintf("/api/groups/%d", id))
	assertStatus(t, rec, http.StatusOK)
	for _, g := range store.ListGroups("") {
		if g.ID == id {
			t.Fatal("group still exists")
		}
	}
}

func TestIntegration_SnippetLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleSnippets, "/api/snippets",
		`{"name":"integ-snip","command":"echo hi"}`)
	assertStatus(t, rec, http.StatusOK)
	idx := int(decodeResp(t, rec)["data"].(map[string]any)["index"].(float64))

	// Verify in list
	rec = getJSON(t, srv.handleSnippets, "/api/snippets")
	assertStatus(t, rec, http.StatusOK)

	rec = putJSON(t, srv.handleSnippetDetail, fmt.Sprintf("/api/snippets/%d", idx),
		`{"name":"integ-snip-v2","command":"echo bye"}`)
	assertStatus(t, rec, http.StatusOK)

	rec = deleteReq(t, srv.handleSnippetDetail, fmt.Sprintf("/api/snippets/%d", idx))
	assertStatus(t, rec, http.StatusOK)
	for _, s := range store.ListSnippets() {
		if s.Name == "integ-snip-v2" {
			t.Fatal("snippet still exists")
		}
	}
}

func TestIntegration_RoleLifecycle(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleRoles, "/api/roles",
		`{"name":"IntegRole","description":"test","system_prompt":"Be precise"}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// List should have 7 builtin + 1 custom
	rec = getJSON(t, srv.handleRoles, "/api/roles")
	assertStatus(t, rec, http.StatusOK)
	if len(decodeResp(t, rec)["data"].([]any)) != 8 {
		t.Fatal("expected 8 roles total")
	}

	rec = putJSON(t, srv.handleRoleDetail, fmt.Sprintf("/api/roles/%d", id),
		`{"name":"EditedRole","description":"edited","system_prompt":"Think carefully"}`)
	assertStatus(t, rec, http.StatusOK)

	rec = deleteReq(t, srv.handleRoleDetail, fmt.Sprintf("/api/roles/%d", id))
	assertStatus(t, rec, http.StatusOK)
	for _, r := range store.ListRoles() {
		if r.ID == id {
			t.Fatal("role still exists")
		}
	}
}

func TestIntegration_TaskEventFlow(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := postJSON(t, srv.handleTaskDetail, "/api/tasks/events",
		`{"task_id":"itask","pane_key":"ipane","event":"user_message","data":{"text":"hi"}}`)
	assertStatus(t, rec, http.StatusOK)
	id := int(decodeResp(t, rec)["data"].(map[string]any)["id"].(float64))

	// Verify total in list
	rec = getJSON(t, srv.handleTasks, "/api/tasks")
	assertStatus(t, rec, http.StatusOK)
	if decodeResp(t, rec)["data"].(map[string]any)["total"] != float64(1) {
		t.Fatal("expected total 1")
	}

	// Complete
	rec = postJSON(t, srv.handleTaskDetail, fmt.Sprintf("/api/tasks/%d/complete", id), "")
	assertStatus(t, rec, http.StatusOK)
	events := store.GetTaskEventsByPane("ipane")
	if len(events) == 0 || !events[0].Completed {
		t.Fatal("task not completed")
	}
}

func TestIntegration_SessionEndpoints(t *testing.T) {
	srv := newTestServerWithDB(t)
	srv.factory = &mockFactory{name: "local command"}

	// GET /api/sessions — empty for local backend
	rec := getJSON(t, srv.handleListSessions, "/api/sessions")
	assertStatus(t, rec, http.StatusOK)

	// GET /api/backends — returns availability
	rec = getJSON(t, srv.handleListBackends, "/api/backends")
	assertStatus(t, rec, http.StatusOK)
	if len(decodeResp(t, rec)["data"].([]any)) != 3 {
		t.Fatal("expected 3 backends")
	}
}

func TestIntegration_AuthEndpoints(t *testing.T) {
	srv := newTestServerWithDB(t)

	rec := getJSON(t, srv.handleAuthCheck, "/api/auth/check")
	assertStatus(t, rec, http.StatusOK)
	resp := decodeResp(t, rec)
	if !resp["success"].(bool) {
		t.Fatal("expected success")
	}
	data := resp["data"].(map[string]any)
	if !data["authenticated"].(bool) {
		t.Fatal("expected authenticated=true")
	}
}

func TestIntegration_PaneStatusFlow(t *testing.T) {
	srv := newTestServerWithDB(t)

	// Post event with pane_key
	postJSON(t, srv.handleTaskDetail, "/api/tasks/events",
		`{"task_id":"ipane-task","pane_key":"ipane-status","event":"msg","data":{"text":"ok"}}`)

	// Set pane status
	rec := putJSON(t, srv.handlePaneStatus, "/api/panes/status",
		`{"pane_key":"ipane-status","status":"running"}`)
	assertStatus(t, rec, http.StatusOK)

	// Verify pane in status list
	rec = getJSON(t, srv.handlePaneStatus, "/api/panes/status")
	assertStatus(t, rec, http.StatusOK)
	data := decodeResp(t, rec)["data"].(map[string]any)
	if data["ipane-status"] != "running" {
		t.Fatalf("expected pane status 'running', got %v", data["ipane-status"])
	}
}
