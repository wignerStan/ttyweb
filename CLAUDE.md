# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ttyweb is a web-based terminal emulator — a Go HTTP/WebSocket server that bridges browser-based terminal sessions to PTY backends. Inspired by gotty. Supports pluggable backends (local command, tmux, zellij). Includes AI integration (OpenAI-compatible LLM, streaming, session scanning), git worktree management via go-git, kanban task board, and multi-tab notepad.

## Build & Run Commands

```bash
# Build frontend (outputs to bindata/static/)
cd frontend && bun install && bun run build

# Build Go binary
go build -o ttyweb .

# Run (requires built frontend)
./ttyweb -w bash                    # local shell backend
./ttyweb -backend tmux -w           # tmux backend
./ttyweb -backend tmux -port 18899  # custom port

# Frontend dev server (proxies API/WS to Go backend on :8080)
cd frontend && bun run dev
```

## Testing

```bash
make test            # Go unit tests + frontend vitest
make test-go         # go test ./... -count=1
make test-frontend   # cd frontend && bunx vitest run
make test-e2e        # Playwright (needs built binary at ./ttyweb)
make test-all        # Everything

# Single Go test
go test ./webtty/... -run TestWebTTY -v

# Single frontend test
cd frontend && bunx vitest run -t "renders sidebar"

# Single E2E test
cd frontend && bunx playwright test -g "auth"

# Go coverage report
make coverage-go     # prints total coverage percentage
```

Coverage gate: 90% minimum (enforced in CI and lefthook pre-push).

## Linting

```bash
make lint            # golangci-lint + biome + tsc
make lint-go         # go vet + golangci-lint run ./...
make lint-frontend   # biome check + tsc --noEmit

# Auto-fix frontend
cd frontend && bun run lint:fix
cd frontend && bun run format
```

golangci-lint v2 config in `.golangci.yml` (25+ linters including gosec, gocyclo, revive). `nolint` requires explanation and specific linter name. Biome handles frontend linting/formatting.

## Git Hooks (lefthook)

Configured in `lefthook.yml`:
- **pre-commit**: go vet, golangci-lint, go test -short, biome check, tsc, vitest (parallel)
- **pre-push**: go test -race, 90% coverage gate

## Architecture

```
main.go                  CLI flags, backend selection, server startup
├── server/              HTTP server, WebSocket handler, REST API, auth
├── webtty/              Core protocol: bridges backend.Slave ↔ WebSocket
├── backend/             Pluggable terminal backends (Factory pattern)
├── ai/                  LLM client, session scanner, conversation parser, streaming
├── db/                  SQLite + GORM (WAL mode, auto-migrate, CGO-free)
├── config/              JSON config + env var overrides
├── worktree/            Git worktree CRUD via go-git (no exec.Command)
├── service/             Business logic: AI sessions, notepad, projects, tasks, worktree ops
├── pkg/                 Internal utilities (validate, homedir, randomstring)
├── frontend/            React SPA (Vite + TypeScript, bun)
└── bindata/static/      Built frontend assets (Go embed target)
```

### Module Details

Each major package has its own `CLAUDE.md` with module-specific patterns:

- **[server/CLAUDE.md](server/CLAUDE.md)** — Server setup, middleware, REST API routing, MemoryStore, API response envelope
- **[backend/CLAUDE.md](backend/CLAUDE.md)** — Factory pattern, Slave/SessionManager interfaces, backend implementations
- **[webtty/CLAUDE.md](webtty/CLAUDE.md)** — WebTTY binary protocol, codecs, master/slave bridge
- **[frontend/CLAUDE.md](frontend/CLAUDE.md)** — React architecture, hooks, mobile views, E2E test setup
- **[ai/CLAUDE.md](ai/CLAUDE.md)** — OpenAI client, session detection/scanning, streaming, STT
- **[db/CLAUDE.md](db/CLAUDE.md)** — SQLite singleton, GORM models, time-partitioned tables
- **[config/CLAUDE.md](config/CLAUDE.md)** — JSON config loading, env var overrides, defaults
- **[worktree/CLAUDE.md](worktree/CLAUDE.md)** — go-git worktree ops, RepoLock, OperationSemaphore
- **[service/CLAUDE.md](service/CLAUDE.md)** — AI sessions, notepad, projects, task segments, worktree service

### Cross-Module Patterns

**Backend Registration** (`main.go`): Backends selected via `-backend` flag in a switch statement. Each implements `backend.Factory`. Server imports as `type Factory = backend.Factory`.

**Embedded Frontend**: `bindata/static/` holds the built React app, served via `go:embed` in `server/server.go`. Vite builds to this directory.

**Input Validation** (`pkg/validate/`): Strict regex allowlist for session names (`^[a-zA-Z0-9_.-]{1,128}$`) and pane IDs (`^[a-zA-Z0-9_.%:-]+$`). Used in API handlers before constructing shell commands.

**Database** (`db/`): Singleton GORM/SQLite with WAL mode. Models use year/month partitioning. `db.Init()` called from `main.go` before server start. Gracefully degrades — server works without DB.

**Configuration** (`config/`): Loads from `~/.config/ttyweb/config.json` (or `-config` flag). Environment variables override JSON values: `LLM_API_KEY`, `LLM_API_URL`, `LLM_MODEL`, `XFYUN_*`, `BUTLER_*`.

**API Response Envelope**: All REST endpoints return `apiResponse{Success, Data, Error}`. Helpers: `writeAPISuccess()`, `writeAPIError()`.

**Service Layer** (`service/`): Business logic between API handlers and data stores. Orchestrates `db`, `ai`, and `worktree` packages. Server handlers call service functions, not data layer directly.

## CLI Flags

```
-config       Path to JSON config file
-addr         IP address (default: 0.0.0.0)
-port         Port (default: 8080)
-path         Base path (default: /)
-backend      Backend: local, tmux, zellij
-credential   Basic auth (deprecated — use TTYWEB_CREDENTIAL env var)
-tls          Enable TLS
-w            Permit client write
-title-format Window title template
-session      Default session name for tmux/zellij
-db           SQLite database path (default: ~/.local/share/ttyweb/ttyweb.db)
```
