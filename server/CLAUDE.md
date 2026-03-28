# CLAUDE.md — server/

HTTP server, WebSocket handler, REST API, middleware, and in-memory data store.

## File Layout

```
server.go          Server struct, HTTP setup, embed.Fs serving, WebSocket origin check
handlers.go        WebSocket connection handler (auth + WebTTY protocol)
api.go             API router (setupAPIHandlers) + session/backend endpoints
api_*.go           Split API handlers by domain:
  api_ai.go        POST /api/ai/command (stub)
  api_auth.go      GET /api/auth/check
  api_butler.go    GET/POST /api/butler/* (stub)
  api_config.go    GET /api/opencode-config
  api_groups.go    CRUD /api/groups
  api_log.go       GET /api/log
  api_panes.go     GET /api/panes/status
  api_profiles.go  CRUD /api/profiles
  api_roles.go     GET /api/roles, /api/roles/defaults
  api_snippets.go  CRUD /api/snippets
  api_tasks.go     CRUD /api/tasks
  api_telemetry.go POST /api/telemetry
  api_tmux.go      /api/tmux/* extensions
  api_upload.go    POST /api/upload
middleware.go      Logger, security headers, basic auth
store.go          MemoryStore — thread-safe in-memory CRUD
slave.go          type Factory = backend.Factory
```

## Key Patterns

**SessionManager Dispatch**: The `sessionManager()` helper returns the active backend's `SessionManager` (or `NoSessionManager`). All session-dependent handlers call this instead of direct type assertion. This decouples `api_*.go` files from specific backends.

**API Response Envelope**: All REST endpoints use `apiResponse{Success, Data, Error}`. Helpers: `writeAPISuccess()`, `writeAPISuccessRaw()`, `writeAPIError()`.

**MemoryStore** (`store.go`): Global singleton (`var store = NewMemoryStore()`) providing thread-safe CRUD for profiles, session groups, snippets, AI roles (7 builtin), task events, and pane statuses. Not persisted across restarts.

**WebSocket Origin Check** (`server.go`): Default `CheckOrigin` parses the `Origin` header (which includes scheme) with `url.Parse()` and compares only the host against `r.Host`.

**Middleware Chain**: `wrapLogger` → `wrapHeaders` (X-Content-Type-Options, X-Frame-Options, Referrer-Policy) → optional `wrapBasicAuth` (constant-time comparison).

## REST API

| Endpoint | Method | Backend | Description |
|----------|--------|---------|-------------|
| `/api/sessions` | GET | tmux/zellij | List sessions |
| `/api/sessions` | POST | tmux/zellij | Create session |
| `/api/sessions/{name}` | GET | tmux/zellij | Session detail |
| `/api/sessions/{name}` | DELETE | tmux/zellij | Kill session |
| `/api/backends` | GET | all | Backend availability |
| `/api/auth/check` | GET | all | Auth status |
| `/api/profiles` | GET/POST | all | Profile CRUD |
| `/api/profiles/{id}` | GET/PUT/DELETE | all | Profile detail |
| `/api/groups` | GET/POST | all | Group CRUD |
| `/api/groups/{id}` | GET/PUT/DELETE | all | Group detail |
| `/api/snippets` | GET/POST | all | Snippet CRUD |
| `/api/snippets/{index}` | GET/PUT/DELETE | all | Snippet detail |
| `/api/roles` | GET | all | AI roles |
| `/api/roles/defaults` | GET | all | Default AI roles |
| `/api/roles/{id}` | GET/PUT/DELETE | all | Role detail |
| `/api/tasks` | GET/POST | all | Task events |
| `/api/tasks/{id}` | GET/DELETE | all | Task detail |
| `/api/panes/status` | GET | all | Pane statuses |
| `/api/ai/command` | POST | all | AI command (stub) |
| `/api/tmux/tree` | GET | tmux/zellij | Session/pane hierarchy |
| `/api/tmux/config` | GET | tmux | Prefix key config |
| `/api/tmux/quick-dirs` | GET | tmux | Quick directories |
| `/api/tmux/new-window` | POST | tmux | New tmux window |
| `/api/tmux/new-session` | POST | tmux | New tmux session |
| `/api/tmux/send-keys` | POST | tmux | Send keys to pane |
| `/api/tmux/pane-mode` | POST | tmux | Toggle pane mode |
| `/api/telemetry` | POST | all | Telemetry events |
| `/api/upload` | POST | all | File upload |
| `/api/log` | GET | all | Server logs |
| `/api/opencode-config` | GET | all | OpenCode config |
| `/api/butler/*` | GET/POST | all | Butler proxy (stub) |

Local backend returns empty arrays or 503 for session-dependent endpoints.
