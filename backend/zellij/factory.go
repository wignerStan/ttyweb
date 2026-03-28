package zellij

import (
	"encoding/json"
	"fmt"

	"ttyweb/backend"
)

// ZellijFactory creates zellij session slaves.
// It implements backend.Factory and backend.SessionManager.
type ZellijFactory struct {
	defaultSession string
}

// NewFactory creates a ZellijFactory with the given default session name.
func NewFactory(defaultSession string) *ZellijFactory {
	return &ZellijFactory{
		defaultSession: defaultSession,
	}
}

func (f *ZellijFactory) Name() string { return "zellij" }

func (f *ZellijFactory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
	session := ""
	if v := params["session"]; len(v) > 0 {
		session = v[0]
	}
	if session == "" {
		session = f.defaultSession
	}

	return NewZellijSlave(session)
}

func (f *ZellijFactory) IsAvailable() bool {
	return IsServerRunning()
}

func (f *ZellijFactory) ListSessions() (json.RawMessage, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}
	return json.Marshal(sessions)
}

func (f *ZellijFactory) CreateSession(name string, command ...string) (string, error) {
	return CreateSession(name, command...)
}

func (f *ZellijFactory) GetSessionDetail(name string) (json.RawMessage, error) {
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
	return json.Marshal(detail)
}

func (f *ZellijFactory) KillSession(name string) error {
	return KillSession(name)
}

// Ensure compile-time interface satisfaction.
var (
	_ backend.Factory        = (*ZellijFactory)(nil)
	_ backend.SessionManager = (*ZellijFactory)(nil)
)
