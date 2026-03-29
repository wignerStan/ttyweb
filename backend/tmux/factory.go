package tmux

import (
	"encoding/json"
	"fmt"

	"ttyweb/backend"
)

// TmuxFactory creates tmux session slaves.
// It implements backend.Factory and backend.SessionManager.
type TmuxFactory struct {
	defaultSession string
}

// NewFactory creates a TmuxFactory with the given default session name.
func NewFactory(defaultSession string) *TmuxFactory {
	return &TmuxFactory{
		defaultSession: defaultSession,
	}
}

func (f *TmuxFactory) Name() string { return "tmux" }

func (f *TmuxFactory) New(params map[string][]string, headers map[string][]string) (backend.Slave, error) {
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

func (f *TmuxFactory) IsAvailable() bool {
	return IsServerRunning()
}

func (f *TmuxFactory) ListSessions() (json.RawMessage, error) {
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

func (f *TmuxFactory) CreateSession(name string, command ...string) (string, error) {
	return CreateSession(name, command...)
}

func (f *TmuxFactory) GetSessionDetail(name string) (json.RawMessage, error) {
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

func (f *TmuxFactory) KillSession(name string) error {
	return KillSession(name)
}

// Ensure compile-time interface satisfaction.
var (
	_ backend.Factory        = (*TmuxFactory)(nil)
	_ backend.SessionManager = (*TmuxFactory)(nil)
)
