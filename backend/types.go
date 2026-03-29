package backend

import (
	"encoding/json"
	"io"

	"ttyweb/webtty"
)

// Slave is webtty.Slave with a Close method.
type Slave interface {
	webtty.Slave
	io.ReadWriter
	Close() error
}

// Factory creates Slave instances for each connection.
type Factory interface {
	Name() string
	New(params map[string][]string, headers map[string][]string) (Slave, error)
}

// SessionManager provides session CRUD operations for multiplexer backends.
// The local backend does not implement this.
type SessionManager interface {
	IsAvailable() bool
	ListSessions() (json.RawMessage, error)
	CreateSession(name string, command ...string) (string, error)
	GetSessionDetail(name string) (json.RawMessage, error)
	KillSession(name string) error
}

// NoSessionManager is returned by backends that don't support session management.
type NoSessionManager struct{}

func (NoSessionManager) IsAvailable() bool                               { return false }
func (NoSessionManager) ListSessions() (json.RawMessage, error)          { return nil, nil }
func (NoSessionManager) CreateSession(string, ...string) (string, error) { return "", ErrNotSupported }
func (NoSessionManager) GetSessionDetail(string) (json.RawMessage, error) {
	return nil, ErrNotSupported
}
func (NoSessionManager) KillSession(string) error { return ErrNotSupported }

// ErrNotSupported indicates the backend doesn't support this operation.
var ErrNotSupported = &backendError{"operation not supported by this backend"}

type backendError struct {
	msg string
}

func (e *backendError) Error() string { return e.msg }
