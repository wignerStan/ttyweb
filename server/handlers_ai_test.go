package server

import (
	"testing"

	"ttyweb/ai"
)

func TestAutoCreateTaskOnWorking(t *testing.T) {
	// Reset the global store for test isolation.
	store = NewMemoryStore()
	sm := ai.NewStateMachine()

	registerAIStateChangeHandlers(sm)

	paneKey := "session1:0:0"

	// Transition to working should create an ai_started_working task event.
	sm.Transition(paneKey, ai.AIStateWorking)

	events := store.GetTaskEventsByPane(paneKey)
	if len(events) != 1 {
		t.Fatalf("expected 1 task event after working transition, got %d", len(events))
	}
	if events[0].Event != "ai_started_working" {
		t.Errorf("expected event ai_started_working, got %s", events[0].Event)
	}
	if events[0].PaneKey != paneKey {
		t.Errorf("expected paneKey %s, got %s", paneKey, events[0].PaneKey)
	}

	// Transition to idle should create an ai_completed task event.
	sm.Transition(paneKey, ai.AIStateIdle)

	events = store.GetTaskEventsByPane(paneKey)
	if len(events) != 2 {
		t.Fatalf("expected 2 task events after idle transition, got %d", len(events))
	}
	if events[1].Event != "ai_completed" {
		t.Errorf("expected event ai_completed, got %s", events[1].Event)
	}
}

func TestAutoCreateTask_NoEventForSameState(t *testing.T) {
	store = NewMemoryStore()
	sm := ai.NewStateMachine()

	registerAIStateChangeHandlers(sm)

	paneKey := "session1:0:0"

	// First transition to working should create an event.
	sm.Transition(paneKey, ai.AIStateWorking)
	events := store.GetTaskEventsByPane(paneKey)
	if len(events) != 1 {
		t.Fatalf("expected 1 event after first working transition, got %d", len(events))
	}

	// Re-transitioning to the same state should NOT create another event.
	sm.Transition(paneKey, ai.AIStateWorking)
	events = store.GetTaskEventsByPane(paneKey)
	if len(events) != 1 {
		t.Fatalf("expected still 1 event after duplicate working transition, got %d", len(events))
	}
}

func TestAutoCreateTask_NoEventForUnrelatedTransitions(t *testing.T) {
	store = NewMemoryStore()
	sm := ai.NewStateMachine()

	registerAIStateChangeHandlers(sm)

	paneKey := "session1:0:0"

	// Transition to idle from unknown should NOT create an event
	// (only working->idle should create ai_completed).
	sm.Transition(paneKey, ai.AIStateIdle)

	events := store.GetTaskEventsByPane(paneKey)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for unknown->idle transition, got %d", len(events))
	}

	// Transition to waiting_approval should NOT create an event.
	sm.Transition(paneKey, ai.AIStateWaitingApproval)

	events = store.GetTaskEventsByPane(paneKey)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for idle->waiting_approval transition, got %d", len(events))
	}
}

func TestAutoCreateTask_MultiplePanes(t *testing.T) {
	store = NewMemoryStore()
	sm := ai.NewStateMachine()

	registerAIStateChangeHandlers(sm)

	// Two different panes transition independently.
	sm.Transition("pane-a", ai.AIStateWorking)
	sm.Transition("pane-b", ai.AIStateWorking)
	sm.Transition("pane-a", ai.AIStateIdle)

	eventsA := store.GetTaskEventsByPane("pane-a")
	eventsB := store.GetTaskEventsByPane("pane-b")

	if len(eventsA) != 2 {
		t.Errorf("expected 2 events for pane-a (started + completed), got %d", len(eventsA))
	}
	if len(eventsB) != 1 {
		t.Errorf("expected 1 event for pane-b (started), got %d", len(eventsB))
	}
}
