package ai

import (
	"sync"
	"testing"
)

func TestStateMachine_InitialUnknown(t *testing.T) {
	sm := NewStateMachine()
	if sm.Get("pane1") != AIStateUnknown {
		t.Errorf("expected unknown, got %s", sm.Get("pane1"))
	}
}

func TestStateMachine_TransitionFiresHandler(t *testing.T) {
	sm := NewStateMachine()
	var got struct {
		sync.Mutex
		paneKey string
		from    AIState
		to      AIState
	}

	sm.OnStateChange(func(paneKey string, from, to AIState) {
		got.Lock()
		defer got.Unlock()
		got.paneKey = paneKey
		got.from = from
		got.to = to
	})

	sm.Transition("pane1", AIStateWorking)

	got.Lock()
	defer got.Unlock()
	if got.paneKey != "pane1" {
		t.Errorf("paneKey: expected pane1, got %s", got.paneKey)
	}
	if got.from != AIStateUnknown {
		t.Errorf("from: expected unknown, got %s", got.from)
	}
	if got.to != AIStateWorking {
		t.Errorf("to: expected working, got %s", got.to)
	}
}

func TestStateMachine_NoDuplicateNotification(t *testing.T) {
	sm := NewStateMachine()
	count := 0
	sm.OnStateChange(func(string, AIState, AIState) { count++ })

	sm.Transition("p1", AIStateWorking)
	sm.Transition("p1", AIStateWorking) // same state -> no notification

	if count != 1 {
		t.Errorf("expected 1 notification, got %d", count)
	}
}

func TestStateMachine_ConcurrentSafe(t *testing.T) {
	sm := NewStateMachine()
	states := []AIState{AIStateIdle, AIStateWorking, AIStateWaitingApproval, AIStateUnknown}
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sm.Transition("pane", states[n%len(states)])
		}(i)
	}
	wg.Wait()
	// Must not panic or deadlock
}

func TestStateMachine_MultipleHandlers(t *testing.T) {
	sm := NewStateMachine()
	var results []string
	var mu sync.Mutex

	sm.OnStateChange(func(pk string, from, to AIState) {
		mu.Lock()
		results = append(results, "h1:"+string(to))
		mu.Unlock()
	})
	sm.OnStateChange(func(pk string, from, to AIState) {
		mu.Lock()
		results = append(results, "h2:"+string(to))
		mu.Unlock()
	})

	sm.Transition("p1", AIStateWaitingApproval)

	mu.Lock()
	defer mu.Unlock()
	if len(results) != 2 {
		t.Fatalf("expected 2 handler calls, got %d", len(results))
	}
	if results[0] != "h1:waiting_approval" || results[1] != "h2:waiting_approval" {
		t.Errorf("expected both handlers called, got %v", results)
	}
}

func TestStateMachine_AllStates(t *testing.T) {
	sm := NewStateMachine()
	sm.Transition("a", AIStateWorking)
	sm.Transition("b", AIStateIdle)
	sm.Transition("c", AIStateWaitingApproval)

	all := sm.AllStates()
	if len(all) != 3 {
		t.Errorf("expected 3 states, got %d", len(all))
	}
	if all["a"] != AIStateWorking {
		t.Errorf("a: expected working, got %s", all["a"])
	}
	if all["b"] != AIStateIdle {
		t.Errorf("b: expected idle, got %s", all["b"])
	}
	if all["c"] != AIStateWaitingApproval {
		t.Errorf("c: expected waiting_approval, got %s", all["c"])
	}
}

func TestStateMachine_AllStatesIsCopy(t *testing.T) {
	sm := NewStateMachine()
	sm.Transition("a", AIStateWorking)

	all := sm.AllStates()
	all["a"] = AIStateIdle // mutate the copy

	if sm.Get("a") != AIStateWorking {
		t.Error("mutating AllStates result should not affect the state machine")
	}
}
