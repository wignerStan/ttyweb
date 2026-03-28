# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ttyweb is a web-based terminal emulator — a Go HTTP/WebSocket server that bridges browser-based terminal sessions to PTY backends. Inspired by gotty. Supports pluggable backends (local command, tmux, zellij).

## Build & Run Commands

```bash
# Build frontend (outputs to bindata/static/)
cd frontend && npm install && npm run build

# Build Go binary
go build -o ttyweb .

# Run (requires built frontend)
./ttyweb -w bash                    # local shell backend
./ttyweb -backend tmux -w           # tmux backend
./ttyweb -backend tmux -port 18899  # custom port

# Frontend dev server (proxies API/WS to Go backend on :8080)
cd frontend && npm run dev
```

## Testing

```bash
make test            # Go unit tests + frontend vitest
make test-go         # go test ./... -v -count=1
make test-frontend   # cd frontend && npx vitest run
make test-e2e        # Playwright (needs built binary at ./ttyweb)
make test-all        # Everything

# Single Go test
go test ./webtty/... -run TestWebTTY -v

# Single frontend test
cd frontend && npx vitest run -t "renders sidebar"

# Single E2E test
cd frontend && npx playwright test -g "auth"
```

## Linting

```bash
make lint            # go vet + tsc --noEmit
make lint-go         # go vet ./...
make lint-frontend   # cd frontend && npx tsc --noEmit
```

## Architecture

```
main.go                  CLI flags, backend selection, server startup
├── server/              HTTP server, WebSocket handler, REST API, auth
├── webtty/              Core protocol: bridges backend.Slave ↔ WebSocket
├── backend/             Pluggable terminal backends (Factory pattern)
│   ├── localcommand/    Raw shell command with PTY
│   ├── tmux/            tmux session attachment/management
│   └── zellij/          zellij session attachment
├── frontend/            React SPA (Vite + TypeScript)
│   └── src/
│       ├── components/  TerminalTab, Sidebar, mobile views
│       ├── hooks/       useWebSocket, useTerminal
│       ├── mobile/      Separate mobile-optimized views
│       └── App.tsx      Routes: / (desktop), /m (mobile)
└── bindata/static/      Built frontend assets (Go embed target)
```

### Key Patterns

**Backend Factory**: Each backend implements `backend.Factory` (creates `backend.Slave` per connection) and `backend.Slave` (webtty.Slave + io.ReadWriter + Close). Multiplexer backends (tmux, zellij) also implement `backend.SessionManager` for REST API session CRUD. The `local` backend uses `backend.NoSessionManager` (returns empty sessions). Backends are registered via switch in `main.go`.

**SessionManager Interface** (`backend/types.go`): Decouples the REST API from specific backends. The server dispatches API calls via `server.factory.(backend.SessionManager)` type assertion. This avoids hardcoding tmux/zellij calls in `server/api.go`.

**WebTTY Protocol**: Custom binary protocol over WebSocket. Single-byte type prefix + base64 payload. Client messages: Input(1), Ping(2), Resize(3), SetEncoding(4). Server messages: Output(1), Pong(2), SetWindowTitle(3), SetReconnect(5), SetBufferSize(6). Defined in `webtty/message_types.go`.

**WebSocket Origin Check**: The server's default `CheckOrigin` compares `Origin` header host against `r.Host`. The `Origin` header includes the scheme (`http://host:port`) so the check parses it with `url.Parse()` before comparing hosts.

**Frontend State**: No global state library. Component-local state with React hooks. WebSocket auto-reconnect built into `useWebSocket` hook.

### REST API (session-managing backends)

- `GET /api/sessions` — list sessions (empty array for `local` backend)
- `POST /api/sessions` — create session (503 for `local` backend)
- `GET /api/sessions/{name}` — session details (503 for `local` backend)
- `DELETE /api/sessions/{name}` — kill session (503 for `local` backend)
- `GET /api/backends` — list backends with availability and active status

## E2E Test Setup

Playwright expects the binary at `./ttyweb` (built from project root). It starts the server on port 18899 with `-backend local -port 18899 -w`. Two projects: `chromium` (desktop) and `mobile` (iPhone viewport). The `reuseExistingServer` flag is enabled outside CI, so a running dev server can be reused. The `make test-e2e` target auto-builds the binary first.
