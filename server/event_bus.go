package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// BusEvent is an event published on the TaskEventBus.
// It uses a string ID and unix timestamp to avoid collision with the
// TaskEvent type in store.go (which has int ID and time.Time).
type BusEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	PaneKey   string         `json:"pane_key,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Timestamp int64          `json:"timestamp"`
}

// TaskEventBus provides a publish/subscribe mechanism for task events.
// Subscribers can listen to pane-specific events or all events globally.
// Publishing is non-blocking: if a subscriber's buffer is full, the event is dropped.
type TaskEventBus struct {
	mu         sync.RWMutex
	paneSubs   map[string][]chan BusEvent // paneKey -> subscriber channels
	globalSubs []chan BusEvent            // global subscriber channels
	closed     bool
}

// NewTaskEventBus creates a new TaskEventBus.
func NewTaskEventBus() *TaskEventBus {
	return &TaskEventBus{
		paneSubs: make(map[string][]chan BusEvent),
	}
}

// Publish sends an event to pane-specific and global subscribers.
// Non-blocking: drops events if subscriber buffers are full.
func (b *TaskEventBus) Publish(event BusEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	// Send to pane-specific subscribers.
	for _, ch := range b.paneSubs[event.PaneKey] {
		select {
		case ch <- event:
		default:
			// Drop event if buffer is full.
		}
	}

	// Send to global subscribers.
	for _, ch := range b.globalSubs {
		select {
		case ch <- event:
		default:
			// Drop event if buffer is full.
		}
	}
}

// Subscribe creates a new subscriber channel for events targeting the given paneKey.
func (b *TaskEventBus) Subscribe(paneKey string) <-chan BusEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan BusEvent, 64)
	b.paneSubs[paneKey] = append(b.paneSubs[paneKey], ch)
	return ch
}

// SubscribeGlobal creates a new subscriber channel that receives all events.
func (b *TaskEventBus) SubscribeGlobal() <-chan BusEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan BusEvent, 64)
	b.globalSubs = append(b.globalSubs, ch)
	return ch
}

// Unsubscribe removes a channel from all subscription lists.
func (b *TaskEventBus) Unsubscribe(ch <-chan BusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove from pane subscriptions.
	for key, subs := range b.paneSubs {
		b.paneSubs[key] = removeChannel(subs, ch)
		if len(b.paneSubs[key]) == 0 {
			delete(b.paneSubs, key)
		}
	}

	// Remove from global subscriptions.
	b.globalSubs = removeChannel(b.globalSubs, ch)
}

// Close drains and closes all subscriber channels.
func (b *TaskEventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true

	closeChannels(b.globalSubs)
	b.globalSubs = nil

	for _, subs := range b.paneSubs {
		closeChannels(subs)
	}
	b.paneSubs = make(map[string][]chan BusEvent)
}

// SubscriberCount returns the total number of active subscribers.
func (b *TaskEventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := len(b.globalSubs)
	for _, subs := range b.paneSubs {
		count += len(subs)
	}
	return count
}

// removeChannel returns a new slice with the given channel removed.
func removeChannel(channels []chan BusEvent, target <-chan BusEvent) []chan BusEvent {
	for i, ch := range channels {
		if ch == target {
			return append(channels[:i], channels[i+1:]...)
		}
	}
	return channels
}

// closeChannels closes all channels, discarding any buffered events.
func closeChannels(channels []chan BusEvent) {
	for _, ch := range channels {
		close(ch)
	}
}

// --- SSE Handler ---

// SSEHandler streams BusEvents to HTTP clients using Server-Sent Events.
type SSEHandler struct {
	bus *TaskEventBus
}

// NewSSEHandler creates a new SSEHandler backed by the given TaskEventBus.
func NewSSEHandler(bus *TaskEventBus) *SSEHandler {
	return &SSEHandler{bus: bus}
}

// ServeHTTP handles SSE connections.
// GET /api/tasks/events/stream?paneKey=<paneKey> for pane-specific events.
// GET /api/tasks/events/stream for all events (global).
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	paneKey := r.URL.Query().Get("paneKey")

	var ch <-chan BusEvent
	if paneKey != "" {
		ch = h.bus.Subscribe(paneKey)
	} else {
		ch = h.bus.SubscribeGlobal()
	}
	defer h.bus.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			if _, err := w.Write([]byte("data: ")); err != nil {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// Ensure BusEvent has a timestamp populated by default.
func newBusEvent(eventType, paneKey string) BusEvent {
	return BusEvent{
		ID:        generateBusEventID(),
		Type:      eventType,
		PaneKey:   paneKey,
		Timestamp: time.Now().Unix(),
	}
}

// busEventCounter is an atomic counter for generating unique BusEvent IDs.
var busEventCounter atomic.Uint64

// generateBusEventID creates a unique ID for a BusEvent.
func generateBusEventID() string {
	n := busEventCounter.Add(1)
	return fmt.Sprintf("evt-%d-%d", time.Now().UnixNano(), n)
}
