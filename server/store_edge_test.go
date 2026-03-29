package server

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryStore_ListProfiles_Empty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	list := s.ListProfiles()
	if len(list) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(list))
	}
}

func TestMemoryStore_ListGroups_Empty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	list := s.ListGroups("")
	if len(list) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(list))
	}
}

func TestMemoryStore_ListGroups_FilterEmpty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	s.CreateGroup(SessionGroup{GroupName: "G1", ProfileKey: "pk1"})
	list := s.ListGroups("pk-nonexistent")
	if len(list) != 0 {
		t.Fatalf("expected 0 groups for nonexistent profile, got %d", len(list))
	}
}

func TestMemoryStore_ListSnippets_Empty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	list := s.ListSnippets()
	if len(list) != 0 {
		t.Fatalf("expected 0 snippets, got %d", len(list))
	}
}

func TestMemoryStore_DeleteSnippet_NotFound(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	err := s.DeleteSnippet(0)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestMemoryStore_ListTasks_Empty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	tasks, total := s.ListTasks(1, 10)
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestMemoryStore_GetTaskEventsByPane_Empty(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	events := s.GetTaskEventsByPane("nonexistent")
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestMemoryStore_ConcurrentWrites(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func(idx int) {
			defer wg.Done()
			s.CreateProfile(Profile{ProfileKey: "pk", Name: "Profile"})
		}(i)
		go func(idx int) {
			defer wg.Done()
			s.CreateGroup(SessionGroup{GroupName: "Group"})
		}(i)
		go func(idx int) {
			defer wg.Done()
			s.CreateSnippet(Snippet{Name: "Snippet", Command: "echo"})
		}(i)
		go func(idx int) {
			defer wg.Done()
			s.SetPaneStatus("pane", "running")
		}(i)
	}
	wg.Wait()

	if len(s.ListProfiles()) != 50 {
		t.Fatalf("expected 50 profiles, got %d", len(s.ListProfiles()))
	}
	if len(s.ListGroups("")) != 50 {
		t.Fatalf("expected 50 groups, got %d", len(s.ListGroups("")))
	}
	if len(s.ListSnippets()) != 50 {
		t.Fatalf("expected 50 snippets, got %d", len(s.ListSnippets()))
	}
}

func TestMemoryStore_ConcurrentReadsAndWrites(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	// Pre-populate.
	s.CreateProfile(Profile{ProfileKey: "pk", Name: "Initial"})
	s.CreateRole(AiRole{Name: "Custom", SystemPrompt: "prompt"})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = s.ListProfiles()
			_ = s.ListRoles()
			_ = s.GetPaneStatuses()
			_ = s.ListSnippets()
		}()
		go func() {
			defer wg.Done()
			s.CreateProfile(Profile{ProfileKey: "pk", Name: "Concurrent"})
			s.SetPaneStatus("p", "running")
		}()
	}
	wg.Wait()
	// Should not panic; just verify counts are reasonable.
	profiles := s.ListProfiles()
	if len(profiles) < 101 {
		t.Fatalf("expected at least 101 profiles, got %d", len(profiles))
	}
}

func TestMemoryStore_ListTasks_PageOne_LimitOne(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "first"})
	s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "second"})
	s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "third"})

	tasks, total := s.ListTasks(1, 1)
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Event != "first" {
		t.Fatalf("expected 'first', got %q", tasks[0].Event)
	}
}

func TestMemoryStore_DeleteSnippet_Reindexes(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	s.CreateSnippet(Snippet{Name: "S1", Command: "a"})
	s.CreateSnippet(Snippet{Name: "S2", Command: "b"})
	s.CreateSnippet(Snippet{Name: "S3", Command: "c"})

	// Delete middle snippet (index 1).
	if err := s.DeleteSnippet(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := s.ListSnippets()
	if len(list) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(list))
	}
	if list[0].Index != 0 {
		t.Fatalf("expected index 0, got %d", list[0].Index)
	}
	if list[1].Index != 1 {
		t.Fatalf("expected index 1, got %d", list[1].Index)
	}
}

func TestMemoryStore_AddTaskEvent_DefaultsID(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	t1 := s.AddTaskEvent(TaskEvent{PaneKey: "p1", Event: "e1"})
	t2 := s.AddTaskEvent(TaskEvent{PaneKey: "p2", Event: "e2"})

	if t2.ID <= t1.ID {
		t.Fatalf("expected t2.ID > t1.ID, got %d <= %d", t2.ID, t1.ID)
	}
}

func TestMemoryStore_CompleteTask_PreservesOtherFields(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	ts := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)
	te := s.AddTaskEvent(TaskEvent{
		PaneKey:   "p1",
		Event:     "msg",
		Timestamp: ts,
	})

	if err := s.CompleteTask(te.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tasks, _ := s.ListTasks(1, 10)
	if !tasks[0].Completed {
		t.Fatal("expected task to be completed")
	}
	if tasks[0].PaneKey != "p1" {
		t.Fatalf("expected pane key p1, got %q", tasks[0].PaneKey)
	}
	if !tasks[0].Timestamp.Equal(ts) {
		t.Fatalf("expected timestamp preserved, got %v", tasks[0].Timestamp)
	}
}
