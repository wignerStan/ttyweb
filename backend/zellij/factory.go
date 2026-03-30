package zellij

import (
	"encoding/json"
	"fmt"

	"ttyweb/backend"
)

// Factory creates zellij session slaves.
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
func (*Factory) Name() string { return "zellij" }

// New creates a new zellij slave instance.
func (f *Factory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	session := ""
	if v := params["session"]; len(v) > 0 {
		session = v[0]
	}
	if session == "" {
		session = f.defaultSession
	}

	return NewZellijSlave(session)
}

// IsAvailable checks whether zellij is installed.
func (*Factory) IsAvailable() bool {
	return IsServerRunning()
}

// ListSessions returns all zellij sessions as JSON.
func (*Factory) ListSessions() (json.RawMessage, error) {
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

// CreateSession creates a new zellij session.
func (*Factory) CreateSession(name string, command ...string) (string, error) {
	return CreateSession(name, command...)
}

// GetSessionDetail returns details for a zellij session.
func (*Factory) GetSessionDetail(name string) (json.RawMessage, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}

	var found *Session
	for i := range sessions {
		if sessions[i].Name == name {
			found = &sessions[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("session %s not found", name)
	}

	detail := SessionDetail{
		Session: *found,
		Panes:   nil, // zellij doesn't expose pane listing via CLI easily
	}
	data, err := json.Marshal(detail)
	if err != nil {
		return nil, fmt.Errorf("GetSessionDetail: marshal: %w", err)
	}
	return data, nil
}

// KillSession terminates a zellij session.
func (*Factory) KillSession(name string) error {
	return KillSession(name)
}

// Ensure compile-time interface satisfaction.
var (
	_ backend.Factory        = (*Factory)(nil)
	_ backend.SessionManager = (*Factory)(nil)
)
