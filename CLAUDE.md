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
├── pkg/                 Internal utilities (validate, homedir, randomstring)
├── frontend/            React SPA (Vite + TypeScript)
└── bindata/static/      Built frontend assets (Go embed target)
```

### Module Details

Each major package has its own `CLAUDE.md` with module-specific patterns and API documentation:

- **[server/CLAUDE.md](server/CLAUDE.md)** — Server setup, middleware, REST API routing, MemoryStore, API response envelope
- **[backend/CLAUDE.md](backend/CLAUDE.md)** — Factory pattern, Slave/SessionManager interfaces, backend implementations
- **[webtty/CLAUDE.md](webtty/CLAUDE.md)** — WebTTY binary protocol, codecs, master/slave bridge
- **[frontend/CLAUDE.md](frontend/CLAUDE.md)** — React architecture, hooks, mobile views, E2E test setup

### Cross-Module Patterns

**Backend Registration** (`main.go`): Backends are selected via `-backend` flag and instantiated with a switch statement. Each must implement `backend.Factory`. The server imports this as `type Factory = backend.Factory`.

**Embedded Frontend**: `bindata/static/` holds the built React app, served via `go:embed` in `server/server.go`. Frontend builds to this directory (`vite.config.ts` → `outDir: '../bindata/static'`).

**Input Validation** (`pkg/validate/`): Strict regex allowlist for session names (`^[a-zA-Z0-9_.-]{1,128}$`) and pane IDs (`^[a-zA-Z0-9_.%:-]+$`). Used in API handlers before constructing shell commands.
