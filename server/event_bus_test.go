package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- BusEvent (TaskEventBus) tests ---

func TestTaskEventBus_PublishSubscribe(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	ch := bus.Subscribe("pane-1")
	defer bus.Unsubscribe(ch)

	event := BusEvent{
		ID:        "evt-1",
		Type:      "task.created",
		PaneKey:   "pane-1",
		Data:      map[string]any{"title": "hello"},
		Timestamp: time.Now().Unix(),
	}
	bus.Publish(event)

	select {
	case got := <-ch:
		if got.ID != "evt-1" {
			t.Errorf("expected event ID evt-1, got %q", got.ID)
		}
		if got.Type != "task.created" {
			t.Errorf("expected type task.created, got %q", got.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestTaskEventBus_GlobalSubscriber(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	ch := bus.SubscribeGlobal()
	defer bus.Unsubscribe(ch)

	event := BusEvent{
		ID:        "evt-g1",
		Type:      "task.completed",
		PaneKey:   "pane-42",
		Timestamp: time.Now().Unix(),
	}
	bus.Publish(event)

	select {
	case got := <-ch:
		if got.ID != "evt-g1" {
			t.Errorf("expected event ID evt-g1, got %q", got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for global event")
	}
}

func TestTaskEventBus_PaneFiltering(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	ch1 := bus.Subscribe("pane-1")
	ch2 := bus.Subscribe("pane-2")
	defer bus.Unsubscribe(ch1)
	defer bus.Unsubscribe(ch2)

	bus.Publish(BusEvent{ID: "evt-p1", PaneKey: "pane-1", Timestamp: time.Now().Unix()})

	select {
	case got := <-ch1:
		if got.ID != "evt-p1" {
			t.Errorf("pane-1: expected evt-p1, got %q", got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("pane-1: timed out waiting for event")
	}

	select {
	case <-ch2:
		t.Error("pane-2 should not have received pane-1 event")
	case <-time.After(50 * time.Millisecond):
		// correct: no event for pane-2
	}
}

func TestTaskEventBus_Unsubscribe(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	ch := bus.Subscribe("pane-1")
	bus.Unsubscribe(ch)

	bus.Publish(BusEvent{ID: "evt-u1", PaneKey: "pane-1", Timestamp: time.Now().Unix()})

	select {
	case <-ch:
		t.Error("should not receive event after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// correct: channel closed or no event
	}
}

func TestTaskEventBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	ch1 := bus.Subscribe("pane-1")
	ch2 := bus.Subscribe("pane-1")
	defer bus.Unsubscribe(ch1)
	defer bus.Unsubscribe(ch2)

	bus.Publish(BusEvent{ID: "evt-m1", PaneKey: "pane-1", Timestamp: time.Now().Unix()})

	for i, ch := range []<-chan BusEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.ID != "evt-m1" {
				t.Errorf("subscriber %d: expected evt-m1, got %q", i, got.ID)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestTaskEventBus_ConcurrentPublish(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	const n = 100
	ch := bus.Subscribe("pane-1")
	defer bus.Unsubscribe(ch)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			bus.Publish(BusEvent{
				ID:        fmt.Sprintf("evt-c%d", i),
				PaneKey:   "pane-1",
				Timestamp: time.Now().Unix(),
			})
		}(i)
	}
	wg.Wait()

	// Publishing is non-blocking and may drop events when the buffer is full.
	// Verify we receive at least some events without deadlock.
	received := 0
	timeout := time.After(2 * time.Second)
	draining := true
	for draining {
		select {
		case <-ch:
			received++
		case <-timeout:
			draining = false
		}
	}
	if received == 0 {
		t.Fatal("expected to receive some events, got 0")
	}
}

// --- SSE Handler tests ---

func TestSSEHandler_StreamsEvents(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()
	handler := NewSSEHandler(bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/tasks/events/stream?paneKey=pane-1", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(rec, req)
	}()

	// Give the handler time to subscribe and start streaming.
	time.Sleep(50 * time.Millisecond)

	bus.Publish(BusEvent{
		ID:        "sse-1",
		Type:      "task.created",
		PaneKey:   "pane-1",
		Data:      map[string]any{"msg": "hello"},
		Timestamp: time.Now().Unix(),
	})

	// Wait for the event to be written.
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-done

	// Verify SSE headers.
	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", ct)
	}
	cc := rec.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", cc)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "data: {") {
		t.Errorf("expected body to contain 'data: {', got: %s", body)
	}
	if !strings.Contains(body, `"id":"sse-1"`) {
		t.Errorf("expected body to contain event JSON, got: %s", body)
	}
}

func TestSSEHandler_GlobalStream(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()
	handler := NewSSEHandler(bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// No paneKey query param -> global subscription.
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/tasks/events/stream", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)

	bus.Publish(BusEvent{
		ID:        "sse-g1",
		Type:      "task.deleted",
		PaneKey:   "pane-99",
		Timestamp: time.Now().Unix(),
	})

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, `"id":"sse-g1"`) {
		t.Errorf("expected global stream to receive event, got: %s", body)
	}
}

func TestSSEHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()
	handler := NewSSEHandler(bus)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/tasks/events/stream", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestSSEHandler_ClientDisconnect(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()
	handler := NewSSEHandler(bus)

	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/tasks/events/stream", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(rec, req)
	}()

	// Cancel immediately to simulate client disconnect.
	cancel()

	select {
	case <-done:
		// correct: handler exited cleanly
	case <-time.After(time.Second):
		t.Fatal("handler did not exit after context cancellation")
	}
}

// --- BusEvent serialization test ---

func TestBusEvent_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	original := BusEvent{
		ID:        "test-123",
		Type:      "task.updated",
		PaneKey:   "pane-5",
		Data:      map[string]any{"key": "value", "num": 42},
		Timestamp: 1710000000,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal BusEvent: %v", err)
	}

	var decoded BusEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal BusEvent: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: expected %q, got %q", original.ID, decoded.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: expected %q, got %q", original.Type, decoded.Type)
	}
	if decoded.PaneKey != original.PaneKey {
		t.Errorf("PaneKey mismatch: expected %q, got %q", original.PaneKey, decoded.PaneKey)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: expected %d, got %d", original.Timestamp, decoded.Timestamp)
	}
}

func TestTaskEventBus_Close_DrainsChannels(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()

	ch := bus.Subscribe("pane-1")
	bus.Close()

	// After close, channel should be closed.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after bus Close()")
	}
}

func TestTaskEventBus_SubscriberCount(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}

	ch1 := bus.Subscribe("pane-1")
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	ch2 := bus.Subscribe("pane-2")
	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(ch1)
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber after unsubscribe, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(ch2)
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestTaskEventBus_Publish_NonBlocking(t *testing.T) {
	t.Parallel()
	bus := NewTaskEventBus()
	defer bus.Close()

	// Create a subscriber but don't consume from it to fill the buffer.
	_ = bus.Subscribe("pane-1")

	// Publish more events than the buffer can hold.
	// This should not block (non-blocking send with drops).
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			bus.Publish(BusEvent{
				ID:        fmt.Sprintf("nb-%d", i),
				PaneKey:   "pane-1",
				Timestamp: time.Now().Unix(),
			})
		}
	}()

	select {
	case <-done:
		// correct: publish completed without blocking
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked when subscriber buffer was full")
	}
}
