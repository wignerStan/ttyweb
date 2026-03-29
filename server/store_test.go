package server

import (
	"sync"
	"testing"
	"time"
)

// --- Profile CRUD ---

func TestMemoryStore_CreateProfile(t *testing.T) {
	s := NewMemoryStore()
	p := s.CreateProfile(Profile{ProfileKey: "pk1", Name: "Profile 1", SortOrder: 1})
	if p.ID != 1 {
		t.Fatalf("expected ID 1, got %d", p.ID)
	}
	if p.Name != "Profile 1" {
		t.Fatalf("expected name 'Profile 1', got %q", p.Name)
	}
}

func TestMemoryStore_ListProfiles(t *testing.T) {
	s := NewMemoryStore()
	s.CreateProfile(Profile{ProfileKey: "pk1", Name: "A"})
	s.CreateProfile(Profile{ProfileKey: "pk2", Name: "B"})
	list := s.ListProfiles()
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}
}

func TestMemoryStore_UpdateProfile(t *testing.T) {
	s := NewMemoryStore()
	created := s.CreateProfile(Profile{ProfileKey: "pk1", Name: "Old"})
	updated, err := s.UpdateProfile(created.ID, Profile{ProfileKey: "pk1", Name: "New"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New" {
		t.Fatalf("expected name 'New', got %q", updated.Name)
	}
}

func TestMemoryStore_UpdateProfile_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.UpdateProfile(999, Profile{Name: "X"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMemoryStore_DeleteProfile(t *testing.T) {
	s := NewMemoryStore()
	created := s.CreateProfile(Profile{ProfileKey: "pk1", Name: "A"})
	err := s.DeleteProfile(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.ListProfiles()) != 0 {
		t.Fatal("expected empty list after delete")
	}
}

func TestMemoryStore_DeleteProfile_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.DeleteProfile(999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// --- Group CRUD ---

func TestMemoryStore_CreateGroup(t *testing.T) {
	s := NewMemoryStore()
	g := s.CreateGroup(SessionGroup{GroupName: "G1", ProfileKey: "pk1"})
	if g.ID != 1 {
		t.Fatalf("expected ID 1, got %d", g.ID)
	}
}

func TestMemoryStore_ListGroups_All(t *testing.T) {
	s := NewMemoryStore()
	s.CreateGroup(SessionGroup{GroupName: "G1", ProfileKey: "pk1"})
	s.CreateGroup(SessionGroup{GroupName: "G2", ProfileKey: "pk2"})
	list := s.ListGroups("")
	if len(list) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(list))
	}
}

func TestMemoryStore_ListGroups_FilterByProfile(t *testing.T) {
	s := NewMemoryStore()
	s.CreateGroup(SessionGroup{GroupName: "G1", ProfileKey: "pk1"})
	s.CreateGroup(SessionGroup{GroupName: "G2", ProfileKey: "pk2"})
	list := s.ListGroups("pk1")
	if len(list) != 1 {
		t.Fatalf("expected 1 group, got %d", len(list))
	}
	if list[0].GroupName != "G1" {
		t.Fatalf("expected G1, got %q", list[0].GroupName)
	}
}

func TestMemoryStore_UpdateGroup(t *testing.T) {
	s := NewMemoryStore()
	created := s.CreateGroup(SessionGroup{GroupName: "G1"})
	updated, err := s.UpdateGroup(created.ID, SessionGroup{GroupName: "G2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.GroupName != "G2" {
		t.Fatalf("expected G2, got %q", updated.GroupName)
	}
}

func TestMemoryStore_UpdateGroup_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.UpdateGroup(999, SessionGroup{GroupName: "X"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMemoryStore_DeleteGroup(t *testing.T) {
	s := NewMemoryStore()
	created := s.CreateGroup(SessionGroup{GroupName: "G1"})
	err := s.DeleteGroup(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.ListGroups("")) != 0 {
		t.Fatal("expected empty list after delete")
	}
}

func TestMemoryStore_DeleteGroup_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.DeleteGroup(999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// --- Snippet CRUD ---

func TestMemoryStore_CreateSnippet(t *testing.T) {
	s := NewMemoryStore()
	sn := s.CreateSnippet(Snippet{Name: "S1", Command: "ls"})
	if sn.Index != 0 {
		t.Fatalf("expected index 0, got %d", sn.Index)
	}
}

func TestMemoryStore_ListSnippets(t *testing.T) {
	s := NewMemoryStore()
	s.CreateSnippet(Snippet{Name: "S1", Command: "ls"})
	s.CreateSnippet(Snippet{Name: "S2", Command: "pwd"})
	list := s.ListSnippets()
	if len(list) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(list))
	}
}

func TestMemoryStore_UpdateSnippet(t *testing.T) {
	s := NewMemoryStore()
	s.CreateSnippet(Snippet{Name: "S1", Command: "ls"})
	updated, err := s.UpdateSnippet(0, Snippet{Name: "S1-updated", Command: "cat"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "S1-updated" {
		t.Fatalf("expected S1-updated, got %q", updated.Name)
	}
}

func TestMemoryStore_UpdateSnippet_OutOfRange(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.UpdateSnippet(5, Snippet{Name: "X"})
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestMemoryStore_DeleteSnippet(t *testing.T) {
	s := NewMemoryStore()
	s.CreateSnippet(Snippet{Name: "S1", Command: "ls"})
	s.CreateSnippet(Snippet{Name: "S2", Command: "pwd"})
	err := s.DeleteSnippet(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list := s.ListSnippets()
	if len(list) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(list))
	}
	// Verify re-indexing: remaining snippet should have index 0.
	if list[0].Index != 0 {
		t.Fatalf("expected re-indexed index 0, got %d", list[0].Index)
	}
}

func TestMemoryStore_DeleteSnippet_OutOfRange(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.UpdateSnippet(-1, Snippet{Name: "X"})
	if err == nil {
		t.Fatal("expected error for negative index")
	}
}

// --- AiRole CRUD ---

func TestMemoryStore_BuiltinRoles(t *testing.T) {
	s := NewMemoryStore()
	roles := s.ListRoles()
	if len(roles) != 7 {
		t.Fatalf("expected 7 builtin roles, got %d", len(roles))
	}
}

func TestMemoryStore_CreateRole(t *testing.T) {
	s := NewMemoryStore()
	r := s.CreateRole(AiRole{Name: "Custom", SystemPrompt: "Be custom"})
	if r.ID != 8 {
		t.Fatalf("expected ID 8, got %d", r.ID)
	}
}

func TestMemoryStore_UpdateRole(t *testing.T) {
	s := NewMemoryStore()
	created := s.CreateRole(AiRole{Name: "Custom", SystemPrompt: "Be custom"})
	updated, err := s.UpdateRole(created.ID, AiRole{Name: "Updated", SystemPrompt: "Be updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatalf("expected Updated, got %q", updated.Name)
	}
}

func TestMemoryStore_UpdateRole_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.UpdateRole(999, AiRole{Name: "X"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMemoryStore_DeleteRole(t *testing.T) {
	s := NewMemoryStore()
	created := s.CreateRole(AiRole{Name: "Custom", SystemPrompt: "Be custom"})
	err := s.DeleteRole(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Builtin roles should remain.
	if len(s.ListRoles()) != 7 {
		t.Fatalf("expected 7 roles after deleting custom, got %d", len(s.ListRoles()))
	}
}

func TestMemoryStore_DeleteRole_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.DeleteRole(999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

// --- Task CRUD ---

func TestMemoryStore_ListTasks_DefaultPagination(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 5; i++ {
		s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "msg"})
	}
	tasks, total := s.ListTasks(0, 0)
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}
	// Default limit is 20, page defaults to 1.
	if len(tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(tasks))
	}
}

func TestMemoryStore_ListTasks_Pagination(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 10; i++ {
		s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "msg"})
	}
	tasks, total := s.ListTasks(2, 3)
	if total != 10 {
		t.Fatalf("expected total 10, got %d", total)
	}
	// Page 2, limit 3: items 3-5 (0-indexed).
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestMemoryStore_ListTasks_PageBeyondRange(t *testing.T) {
	s := NewMemoryStore()
	tasks, total := s.ListTasks(100, 10)
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestMemoryStore_AddTaskEvent_SetsTimestamp(t *testing.T) {
	s := NewMemoryStore()
	te := s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "msg"})
	if te.ID != 1 {
		t.Fatalf("expected ID 1, got %d", te.ID)
	}
	if te.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestMemoryStore_AddTaskEvent_PreservesTimestamp(t *testing.T) {
	s := NewMemoryStore()
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	te := s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "msg", Timestamp: ts})
	if !te.Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp %v, got %v", ts, te.Timestamp)
	}
}

func TestMemoryStore_CompleteTask(t *testing.T) {
	s := NewMemoryStore()
	te := s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "msg"})
	err := s.CompleteTask(te.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tasks, _ := s.ListTasks(1, 10)
	if !tasks[0].Completed {
		t.Fatal("expected task to be completed")
	}
}

func TestMemoryStore_CompleteTask_NotFound(t *testing.T) {
	s := NewMemoryStore()
	err := s.CompleteTask(999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMemoryStore_GetTaskEventsByPane(t *testing.T) {
	s := NewMemoryStore()
	s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "e1"})
	s.AddTaskEvent(TaskEvent{PaneKey: "p2", Event: "e2"})
	s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "e3"})
	events := s.GetTaskEventsByPane("p1")
	if len(events) != 2 {
		t.Fatalf("expected 2 events for p1, got %d", len(events))
	}
}

// --- Pane Status ---

func TestMemoryStore_SetPaneStatus(t *testing.T) {
	s := NewMemoryStore()
	s.SetPaneStatus("pane1", "running")
	statuses := s.GetPaneStatuses()
	if statuses["pane1"] != "running" {
		t.Fatalf("expected 'running', got %q", statuses["pane1"])
	}
}

func TestMemoryStore_GetPaneStatuses_Empty(t *testing.T) {
	s := NewMemoryStore()
	statuses := s.GetPaneStatuses()
	if len(statuses) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(statuses))
	}
}

func TestMemoryStore_GetPaneStatuses_Copy(t *testing.T) {
	s := NewMemoryStore()
	s.SetPaneStatus("p1", "running")
	original := s.GetPaneStatuses()
	s.SetPaneStatus("p2", "stopped")
	// The returned map should be a copy, not affected by subsequent writes.
	if _, ok := original["p2"]; ok {
		t.Fatal("GetPaneStatuses should return a copy")
	}
}

// --- Thread Safety ---

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.CreateProfile(Profile{ProfileKey: "pk", Name: "Concurrent"})
		}()
		go func() {
			defer wg.Done()
			_ = s.ListProfiles()
		}()
	}
	wg.Wait()
	if len(s.ListProfiles()) != 100 {
		t.Fatalf("expected 100 profiles, got %d", len(s.ListProfiles()))
	}
}
