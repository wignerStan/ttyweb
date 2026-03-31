# Changelog

All notable changes to ttyweb will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Core terminal emulation with xterm.js and WebSocket PTY bridge (#1)
- Pluggable backend system with Factory pattern: local shell, tmux, zellij (#1)
- HTTP server with `go:embed` frontend assets, WebSocket handler, and middleware chain (#1)
- REST API with SessionManager dispatch pattern and `apiResponse` envelope (#1)
- Basic authentication with constant-time credential comparison (#1)
- Input validation with strict regex allowlist for session names and pane IDs (#1)
- MemoryStore for in-memory CRUD (profiles, groups, snippets, AI roles, task events, pane statuses) (#1)
- JSON configuration system (`~/.config/ttyweb/config.json`) with environment variable overrides: `LLM_API_KEY`, `LLM_API_URL`, `LLM_MODEL`, `XFYUN_*`, `BUTLER_*` (#6)
- SQLite database with GORM (WAL mode, auto-migration, CGO-free via `glebarez/sqlite`) (#16)
- Xunfei STT WebSocket proxy for voice input (`/ws/speech`) with HMAC-SHA256 auth (#3)
- OpenAI-compatible command generation with 7 built-in roles and markdown fence extraction (#7)
- Streaming LLM responses via WebSocket (`/ws/ai/stream`) with role selector UI (#18)
- Claude Code and Codex session detection from command-line patterns (#9)
- AI session scanner with JSONL conversation parsing (messages, tool use blocks) (#9)
- Imperial Study orchestration dashboard (22 components): floating panel, workers, inbox, activity feed, assistant chat, run pipeline, command input, task detail modal (#2)
- Butler reverse proxy with SSE passthrough, path validation, and body limits (#8)
- Kanban task board with drag-and-drop (Todo, In Progress, Done, Archived columns) via `@dnd-kit` (#5)
- Task-AI session many-to-many linking (#4)
- Persistent task tracking with segments, chat messages, command records, and summaries (#13)
- Git worktree management with go-git (create, list, sync, delete, commit, refresh status) (#12)
- Per-repo operation locks (`RepoLock`) and bounded semaphore (`OperationSemaphore`) for git concurrency control (#32)
- Project workspace management with git directory validation (#1)
- Multi-tab notepad with auto-save, reorder, and project association (#11, #17)
- AI conversation history viewer with grouped list, filter, and expandable tool results (#19)
- Project and worktree management UI (project list, worktree cards, status badges, create dialog) (#14)
- Tmux extensions: tree view, new session/window, send keys, prefix config (#1)
- File upload endpoint (10MB limit) (#1)
- Health check endpoint (`GET /api/health`) for load balancer probes (#41)
- Server log endpoint (last 1000 lines) (#1)
- Telemetry endpoint (#1)
- Mobile-optimized UI (`/m` route) with touch toolbox, font size slider, keyboard mode toggle, iOS viewport fix, and shake-to-record (#1)
- CI pipeline: Dependabot (gomod, npm, github-actions), golangci-lint v2, govulncheck, Playwright E2E with artifact upload on failure (#41)
- 91 Go test files and 99 frontend test files

### Changed
- Replaced `exec.Command` git CLI calls with native go-git library for worktree operations; removed 16 dead functions (#57)
- Promoted `golang.org/x/time` to direct dependency, bumped Go to 1.25 (#41)
- Migrated `.golangci.yml` to v2 config schema (#38)
- Replaced global gosec exclusions with per-line suppressions; added 10 new linters (gosec, gocyclo, gocognit, nestif, gocritic, revive, nolintlint, wrapcheck, rowserrcheck, sqlclosecheck) (#37)
- Improved biome linter config: enabled a11y rules, type safety, removed non-null assertion override (#40)
- Wired graceful shutdown to signal context (SIGINT/SIGTERM triggers `srv.Shutdown`) (#35)
- Moved rate limiter cleanup to background goroutine (#41)

### Deprecated
- `-credential` CLI flag in favor of `TTYWEB_CREDENTIAL` environment variable (#41)

### Removed
- 16 dead functions: porcelain parsers, git CLI wrappers, `buildGitCommandEnv`, `runGitCmd*`, `parseWorktreeList`, and 27 associated test functions (#57)
- G702 global gosec exclusion from `.golangci.yml` (#57)

### Fixed
- Sanitized ~60 `err.Error()` leaks across all API and WebSocket handlers to prevent information leakage (#41)
- Fixed production data race in `Server.Run()` — goroutine sharing `err` variable with main goroutine (#34)
- Added PTY mutex protection (`sync.Mutex`) to all 3 backend implementations against concurrent `Close()` vs `Read`/`Write`/`ResizeTerminal` (#34)
- Fixed RepoLock double-release deadlock via `sync.Once` (#32)
- Made `RepoLock.Lock()` and `RLock` respect context cancellation via `TryLock` spin loop (#32)
- Added `RepoLock.Remove()` for bounded map growth (#32)
- Held service lock during `syncWorktrees` git I/O to prevent TOCTOU race (#32)
- Sanitized git environment variables (`GIT_DIR`, `GIT_WORK_TREE`, etc.) in all test helpers and production code to prevent ghost commits (#32)
- Serialized commits on same worktree to prevent `index.lock` race (#32)
- Resolved `titleVariables` panic on missing format variables (#41)
- Returned HTTP 503 when max connections exceeded instead of silently dropping requests (#35)
- Fixed WebSocket origin check comparing full URL against bare host (#1)
- Fixed double-close panic and path encoding collision in routes (#21)
- Validated AI streaming URLs and added auth check to prevent SSRF (#23)
- Whitelisted updatable project fields and removed `log.Fatalf`/`panic` calls (#27, #28)
- Hardened Butler proxy with path validation, body limits, and context propagation (#20)
- Fixed 208 TypeScript errors in test mocks (untyped `vi.fn()`, `noUncheckedIndexedAccess`, DOM query narrowing) (#36)
- Used `git init -b main` in test helpers for CI compatibility (#51)
- Mocked `toLocaleTimeString` for timezone-safe snapshot tests (#51)
- Bumped `cloudflare/circl` to v1.6.3 (GO-2026-4550) (#41)
- Allowed microphone in `Permissions-Policy` (#1)
- Addressed bot review feedback across multiple PRs (#40, #42)
- Fixed pre-existing bug in `RefreshWorktree` path resolution (#57)

### Security
- Added per-IP rate limiting middleware (10 req/s, burst 20) using `golang.org/x/time/rate` (#41)
- Added CORS middleware with default deny-all cross-origin policy, configurable via `--cors-origins` (#41)
- Added CSRF protection requiring `X-Requested-With: XMLHttpRequest` on mutating requests (#41)
- Added security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`) (#1)
- Added `ReadHeaderTimeout` to HTTP server (gosec G112) (#37)
- Added bounds checking for slice access (gosec G602) and `validate.Uint16()` for PTY resize (gosec G115) (#37)
- Tightened `MkdirAll` permissions to 0750 (gosec G301) (#37)
- Sanitized internal errors across all API handlers and WebSocket handlers (#41)
- Validated AI streaming URLs to prevent SSRF via client-supplied `apiUrl` (#23)
- Hardened Butler proxy with path traversal prevention and 1MB body limit (#20)
- Whitelisted updatable project fields to prevent mass assignment (#28)
- Replaced global gosec exclusions with targeted per-line suppressions (#37)
- Added `govulncheck` to CI pipeline for automated vulnerability scanning (#41)
