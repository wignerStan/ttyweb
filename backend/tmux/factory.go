package tmux

import (
	"encoding/json"
	"fmt"

	"ttyweb/backend"
)

// Factory creates tmux session slaves.
// It implements backend.Factory and backend.SessionManager.
type Factory struct {
	defaultSession string
}

// NewFactory creates a Factory with the given default session name.
func NewFactory(defaultSession string) *Factory {
	return &Factory{
		defaultSession: defaultSession,
	}
}

// Name returns the backend name.
func (_ *Factory) Name() string { return "tmux" }

// New creates a new tmux slave instance.
func (f *Factory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	session := ""
	if v := params["session"]; len(v) > 0 {
		session = v[0]
	}
	if session == "" {
		session = f.defaultSession
	}

	pane := ""
	if v := params["pane"]; len(v) > 0 {
		pane = v[0]
	}

	return NewTmuxSlave(session, pane)
}

// IsAvailable checks whether tmux is installed and running.
func (_ *Factory) IsAvailable() bool {
	return IsServerRunning()
}

// ListSessions returns all tmux sessions as JSON.
func (_ *Factory) ListSessions() (json.RawMessage, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(sessions)
	if err != nil {
		return nil, fmt.Errorf("ListSessions: marshal: %w", err)
	}
	return data, nil
}

// CreateSession creates a new tmux session.
func (_ *Factory) CreateSession(name string, command ...string) (string, error) {
	return CreateSession(name, command...)
}

// GetSessionDetail returns details for a tmux session.
func (_ *Factory) GetSessionDetail(name string) (json.RawMessage, error) {
	detail, err := GetSessionDetail(name)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(detail)
	if err != nil {
		return nil, fmt.Errorf("GetSessionDetail: marshal: %w", err)
	}
	return data, nil
}

// KillSession terminates a tmux session.
func (_ *Factory) KillSession(name string) error {
	return KillSession(name)
}

// Ensure compile-time interface satisfaction.
var (
	_ backend.Factory        = (*Factory)(nil)
	_ backend.SessionManager = (*Factory)(nil)
)
