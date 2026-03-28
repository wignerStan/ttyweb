# CLAUDE.md — backend/

Pluggable terminal backends using the Factory pattern. Each backend implements `Factory` (creates `Slave` per connection) and `Slave` (webtty.Slave + io.ReadWriter + Close).

## Core Interfaces (`types.go`)

```go
Factory        → Name() string, New(params, headers) (Slave, error)
Slave          → webtty.Slave + io.ReadWriter + Close()
SessionManager → IsAvailable(), ListSessions(), CreateSession(), GetSessionDetail(), KillSession()
NoSessionManager → Stub returning false/nil/ErrNotSupported
```

Multiplexer backends (tmux, zellij) implement both `Factory` and `SessionManager`. The local backend uses `NoSessionManager`.

## Backend Implementations

| Directory | Backend | SessionManager | Notes |
|-----------|---------|----------------|-------|
| `localcommand/` | local | No | Raw shell command with PTY via `creack/pty` |
| `tmux/` | tmux | Yes | Full session+pane management, `tmux` CLI |
| `zellij/` | zellij | Yes | Session management only (no panes), `zellij` CLI |

## Registration

Backends are selected in `main.go` via `-backend` flag with a switch statement. Adding a new backend requires:
1. Implement `backend.Factory` and `backend.Slave`
2. Optionally implement `backend.SessionManager`
3. Add a case in `main.go`'s switch

## Validation

Session names and pane IDs from API paths are validated by `pkg/validate` before being passed to backends. Backends receive pre-validated strings — do not re-validate.

## Testing

```bash
go test ./backend/... -v -count=1
go test ./backend/localcommand/... -run TestLocalCommand -v
go test ./backend/tmux/... -run TestTmuxSessions -v
go test ./backend/zellij/... -run TestZellijSessions -v
```
