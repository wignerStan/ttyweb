package server

import (
	"strings"
	"testing"
)

// TestNewBusEvent tests that newBusEvent creates an event with required fields.
func TestNewBusEvent(t *testing.T) {
	evt := newBusEvent("test-event", "pane-1")

	if evt.ID == "" {
		t.Error("expected non-empty ID")
	}
	if evt.Type != "test-event" {
		t.Errorf("expected type 'test-event', got %q", evt.Type)
	}
	if evt.PaneKey != "pane-1" {
		t.Errorf("expected paneKey 'pane-1', got %q", evt.PaneKey)
	}
	if evt.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}
	// ID should have the expected prefix.
	if !strings.HasPrefix(evt.ID, "evt-") {
		t.Errorf("expected ID to start with 'evt-', got %q", evt.ID)
	}
}

// TestGenerateBusEventID tests that generated IDs are unique and well-formed.
func TestGenerateBusEventID(t *testing.T) {
	id1 := generateBusEventID()
	id2 := generateBusEventID()

	if id1 == id2 {
		t.Error("expected unique IDs")
	}
	if !strings.HasPrefix(id1, "evt-") {
		t.Errorf("expected ID to start with 'evt-', got %q", id1)
	}
	if !strings.HasPrefix(id2, "evt-") {
		t.Errorf("expected ID to start with 'evt-', got %q", id2)
	}
}

// TestTaskEventBus_PublishAfterClose tests that Publish is a no-op after Close.
func TestTaskEventBus_PublishAfterClose(t *testing.T) {
	bus := NewTaskEventBus()
	ch := bus.Subscribe("pane-1")
	bus.Close()

	// Publishing after close should not panic or send.
	bus.Publish(BusEvent{ID: "after-close", PaneKey: "pane-1", Timestamp: 0})

	// Channel should be closed.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after Close()")
	}
}
