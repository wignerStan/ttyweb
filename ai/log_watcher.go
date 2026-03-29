package ai

import (
	"context"
	"sync"
	"time"
)

// SessionEventType describes the type of change detected in a session file.
type SessionEventType string

const (
	// SessionEventNew indicates a newly discovered session.
	SessionEventNew SessionEventType = "new"
	// SessionEventUpdated indicates a session file has been modified.
	SessionEventUpdated SessionEventType = "updated"
	// SessionEventCompleted indicates a session file has been removed.
	SessionEventCompleted SessionEventType = "completed"
)

// SessionEvent represents a detected change in an AI session file.
type SessionEvent struct {
	Type      SessionEventType
	Session   Session
	Timestamp time.Time
}

// LogWatcher monitors AI assistant session files for changes.
// It polls session files at a configurable interval and emits events
// when new sessions are discovered, sessions are updated, or completed.
type LogWatcher struct {
	mu          sync.Mutex
	projectPath string
	interval    time.Duration
	sessions    map[string]*Session // sessionID -> last known state
	events      chan SessionEvent
	cancel      context.CancelFunc
	done        chan struct{}
	started     bool // true after Watch() is called
	stopped     bool // true after Stop() closes the events channel
}

// NewLogWatcher creates a new LogWatcher for the given project path.
// If projectPath is empty, it watches Codex sessions globally.
func NewLogWatcher(projectPath string, interval time.Duration) *LogWatcher {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	return &LogWatcher{
		projectPath: projectPath,
		interval:    interval,
		sessions:    make(map[string]*Session),
		events:      make(chan SessionEvent, 32),
		done:        make(chan struct{}),
	}
}

// Watch starts polling session files and returns a channel of events.
// The caller should call Stop to clean up resources.
func (w *LogWatcher) Watch() <-chan SessionEvent {
	ctx, cancel := context.WithCancel(context.Background())

	w.mu.Lock()
	w.cancel = cancel
	w.started = true
	w.mu.Unlock()

	go w.pollLoop(ctx)

	return w.events
}

// Stop terminates the polling goroutine and closes the event channel.
// Safe to call multiple times.
func (w *LogWatcher) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	wasStarted := w.started
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()

	if wasStarted {
		<-w.done
	}

	close(w.events)
}

// pollLoop runs the polling loop until context is cancelled.
func (w *LogWatcher) pollLoop(ctx context.Context) {
	defer close(w.done)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Initial scan.
	w.scan()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan()
		}
	}
}

// scan performs a single scan pass and emits events for changes.
func (w *LogWatcher) scan() {
	if w.projectPath != "" {
		w.scanClaude()
	}
	w.scanCodex()
}

func (w *LogWatcher) scanClaude() {
	current, err := ScanClaudeSessions(w.projectPath)
	if err != nil {
		return
	}
	w.emitSessionChanges(current, "")
}

func (w *LogWatcher) scanCodex() {
	current, err := ScanCodexSessions()
	if err != nil {
		return
	}
	w.emitSessionChanges(current, string(AssistantTypeCodex))
}

// emitSessionChanges compares the current scan results against tracked sessions,
// emitting events for new, updated, and completed sessions.
// The completedFilter param restricts completed-event detection to sessions of that type;
// an empty string means detect completions for all session types.
func (w *LogWatcher) emitSessionChanges(current []Session, completedFilter string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	currentMap := make(map[string]*Session, len(current))
	for i := range current {
		session := &current[i]
		sessionID := session.SessionID
		currentMap[sessionID] = session

		prev, exists := w.sessions[sessionID]
		if !exists {
			w.sendEvent(SessionEventNew, session)
		} else if session.FileModTime.After(prev.FileModTime) || session.FileSize != prev.FileSize {
			w.sendEvent(SessionEventUpdated, session)
		}
	}

	// Detect completed sessions (files no longer present).
	for sessionID, prev := range w.sessions {
		if _, exists := currentMap[sessionID]; !exists {
			if completedFilter == "" || prev.Type == completedFilter {
				w.sendEvent(SessionEventCompleted, prev)
				delete(w.sessions, sessionID)
			}
		}
	}

	for sessionID, session := range currentMap {
		cp := *session
		w.sessions[sessionID] = &cp
	}
}

// sendEvent emits a session event, dropping it if the channel is full.
// Must be called with w.mu held.
func (w *LogWatcher) sendEvent(eventType SessionEventType, session *Session) {
	select {
	case w.events <- SessionEvent{
		Type:      eventType,
		Session:   *session,
		Timestamp: time.Now(),
	}:
	default:
	}
}

// GetSessions returns a snapshot of currently tracked sessions.
func (w *LogWatcher) GetSessions() []Session {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]Session, 0, len(w.sessions))
	for _, session := range w.sessions {
		result = append(result, *session)
	}
	return result
}
