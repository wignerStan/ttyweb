# ttyweb

A web-based terminal emulator — Go HTTP/WebSocket server bridging browser terminals to PTY backends. Inspired by [gotty](https://github.com/yudai/gotty).

## Features

- **Pluggable backends** — local shell, tmux, zellij (Factory pattern)
- **WebTTY protocol** — binary WebSocket protocol with base64 encoding
- **REST API** — session management, profiles, groups, snippets, tasks, file upload
- **AI integration** — OpenAI-compatible LLM client, streaming, session scanning (Claude Code/Codex detection)
- **Git worktree management** — create/list/delete/commit worktrees via native go-git
- **Kanban task board** — drag-and-drop columns (Todo, In Progress, Done, Archived)
- **Multi-tab notepad** — global and project-scoped notes with auto-save
- **Mobile UI** — optimized layout at `/m` with touch toolbox, voice input, shake-to-record
- **Basic auth** — with constant-time credential comparison
- **TLS support** — optional HTTPS
- **SQLite persistence** — GORM with WAL mode, CGO-free
- **JSON configuration** — `~/.config/ttyweb/config.json` with env var overrides

## Quick Start

```bash
# Install frontend dependencies and build
cd frontend && bun install && bun run build && cd ..

# Build Go binary
go build -o ttyweb .

# Run with a local shell
./ttyweb -w bash

# Run with tmux backend
./ttyweb -backend tmux -w
```

Open `http://localhost:8080` in a browser.

## Development

### Frontend Dev Server

```bash
# Start Go backend on :8080
./ttyweb -w bash &

# Start Vite dev server (proxies /api and /ws to Go backend)
cd frontend && bun run dev
```

Open `http://localhost:5173` for hot-reloading frontend development.

### Testing

```bash
make test          # Go unit tests + frontend vitest
make test-e2e      # Playwright E2E (needs built binary)
make test-all      # Everything
make coverage-go   # Go coverage percentage
```

### Linting

```bash
make lint          # golangci-lint + biome + tsc
```

## CLI Flags

```
-config         Path to JSON config file
-addr           IP address (default: 0.0.0.0)
-port           Port (default: 8080)
-path           Base path (default: /)
-backend        Backend: local, tmux, zellij
-credential     Basic auth user:pass (deprecated — use TTYWEB_CREDENTIAL env var)
-tls            Enable TLS
-tls-crt        TLS certificate file
-tls-key        TLS key file
-w              Permit client write
-title-format   Window title format (Go template)
-session        Default session name for tmux/zellij
-db             SQLite database path
```

## Configuration

Create `~/.config/ttyweb/config.json`:

```json
{
  "llm": {
    "apiKey": "sk-...",
    "apiUrl": "https://api.openai.com/v1/chat/completions",
    "model": "gpt-4o"
  },
  "xfyun": {
    "appId": "...",
    "apiKey": "...",
    "apiSecret": "..."
  },
  "butler": {
    "host": "localhost",
    "port": "8215"
  }
}
```

Environment variables override JSON values: `LLM_API_KEY`, `LLM_API_URL`, `LLM_MODEL`, `XFYUN_APP_ID`, `XFYUN_API_KEY`, `XFYUN_API_SECRET`, `BUTLER_HOST`, `BUTLER_PORT`.

## Architecture

```
main.go           CLI flags, backend selection, server startup
├── server/       HTTP server, WebSocket handler, REST API (50+ endpoints)
├── webtty/       Binary protocol: backend.Slave ↔ WebSocket bridge
├── backend/      Pluggable backends: local, tmux, zellij
├── ai/           LLM client, session scanner, streaming, STT
├── db/           SQLite + GORM (WAL, auto-migrate)
├── config/       JSON config + env var overrides
├── worktree/     Git worktree CRUD via go-git
├── service/      Business logic layer
├── pkg/          Utilities (validate, homedir, randomstring)
├── frontend/     React SPA (Vite + TypeScript)
└── bindata/      Embedded frontend assets
```

## Tech Stack

**Backend**: Go 1.25, gorilla/websocket, go-git, GORM, SQLite (glebarez/sqlite — CGO-free)

**Frontend**: React 19, TypeScript, Vite, xterm.js, @dnd-kit, Biome

**Testing**: Go testing, Vitest, Playwright

**Linting**: golangci-lint v2 (25+ linters), Biome, TypeScript strict mode

**CI**: GitHub Actions — go vet, golangci-lint, govulncheck, race detector, 90% coverage gate, E2E burn-in (3x)
