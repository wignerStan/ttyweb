package ai

import (
	"context"
	"os"
	"sync"
	"time"
)

// AISessionEventType describes the type of change detected in a session file.
type AISessionEventType string

const (
	AISessionEventNew      AISessionEventType = "new"
	AISessionEventUpdated  AISessionEventType = "updated"
	AISessionEventCompleted AISessionEventType = "completed"
)

// AISessionEvent represents a detected change in an AI session file.
type AISessionEvent struct {
	Type      AISessionEventType
	Session   AISession
	Timestamp time.Time
}

// LogWatcher monitors AI assistant session files for changes.
// It polls session files at a configurable interval and emits events
// when new sessions are discovered, sessions are updated, or completed.
type LogWatcher struct {
	mu          sync.Mutex
	projectPath string
	interval    time.Duration
	sessions    map[string]AISession // sessionID -> last known state
	events      chan AISessionEvent
	cancel      context.CancelFunc
	done        chan struct{}
	started     bool // true after Watch() is called
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
		sessions:    make(map[string]AISession),
		events:      make(chan AISessionEvent, 32),
		done:        make(chan struct{}),
	}
}

// Watch starts polling session files and returns a channel of events.
// The caller should call Stop to clean up resources.
func (w *LogWatcher) Watch() <-chan AISessionEvent {
	ctx, cancel := context.WithCancel(context.Background())

	w.mu.Lock()
	w.cancel = cancel
	w.started = true
	w.mu.Unlock()

	go w.pollLoop(ctx)

	return w.events
}

// Stop terminates the polling goroutine and closes the event channel.
func (w *LogWatcher) Stop() {
	w.mu.Lock()
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

	w.mu.Lock()
	defer w.mu.Unlock()

	currentMap := make(map[string]AISession, len(current))
	for _, session := range current {
		sessionID := session.SessionID
		currentMap[sessionID] = session

		prev, exists := w.sessions[sessionID]
		if !exists {
			// New session.
			select {
			case w.events <- AISessionEvent{
				Type:      AISessionEventNew,
				Session:   session,
				Timestamp: time.Now(),
			}:
			default:
			}
		} else if session.FileModTime.After(prev.FileModTime) || session.FileSize != prev.FileSize {
			// Updated session.
			select {
			case w.events <- AISessionEvent{
				Type:      AISessionEventUpdated,
				Session:   session,
				Timestamp: time.Now(),
			}:
			default:
			}
		}
	}

	// Detect completed sessions (files no longer present).
	for sessionID, prev := range w.sessions {
		if _, exists := currentMap[sessionID]; !exists {
			select {
			case w.events <- AISessionEvent{
				Type:      AISessionEventCompleted,
				Session:   prev,
				Timestamp: time.Now(),
			}:
			default:
			}
			delete(w.sessions, sessionID)
		}
	}

	// Update tracked sessions.
	for sessionID, session := range currentMap {
		w.sessions[sessionID] = session
	}
}

func (w *LogWatcher) scanCodex() {
	current, err := ScanCodexSessions()
	if err != nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	currentMap := make(map[string]AISession, len(current))
	for _, session := range current {
		sessionID := session.SessionID
		currentMap[sessionID] = session

		prev, exists := w.sessions[sessionID]
		if !exists {
			select {
			case w.events <- AISessionEvent{
				Type:      AISessionEventNew,
				Session:   session,
				Timestamp: time.Now(),
			}:
			default:
			}
		} else if session.FileModTime.After(prev.FileModTime) || session.FileSize != prev.FileSize {
			select {
			case w.events <- AISessionEvent{
				Type:      AISessionEventUpdated,
				Session:   session,
				Timestamp: time.Now(),
			}:
			default:
			}
		}
	}

	// Detect completed sessions.
	for sessionID, prev := range w.sessions {
		if _, exists := currentMap[sessionID]; !exists && prev.Type == string(AssistantTypeCodex) {
			select {
			case w.events <- AISessionEvent{
				Type:      AISessionEventCompleted,
				Session:   prev,
				Timestamp: time.Now(),
			}:
			default:
			}
			delete(w.sessions, sessionID)
		}
	}

	for sessionID, session := range currentMap {
		w.sessions[sessionID] = session
	}
}

// GetSessions returns a snapshot of currently tracked sessions.
func (w *LogWatcher) GetSessions() []AISession {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]AISession, 0, len(w.sessions))
	for _, session := range w.sessions {
		result = append(result, session)
	}
	return result
}

// isDirWritable checks if a directory path exists and is accessible.
func isDirWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
