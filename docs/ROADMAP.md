# ttyweb Feature Roadmap

What's done, what's not done, and references for inspiration.

Reference projects: [CodeKanban](https://github.com/fy0/CodeKanban), [opencode-tmuxweb](https://github.com/includewudi/opencode-tmuxweb), [worktrunk](https://github.com/max-sixty/worktrunk).

Legend: **Done** = fully implemented | **Partial** = structure exists, gaps remain | **Stub** = returns hardcoded/empty response | **Missing** = no implementation

---

## 1. Core Terminal

| Feature | Status | Notes |
|---------|--------|-------|
| xterm.js terminal emulation | **Done** | |
| WebSocket PTY bridge | **Done** | |
| Local shell backend | **Done** | |
| Tmux backend (session/pane CRUD) | **Done** | |
| Zellij backend | **Done** | |
| Tmux control mode (single PTY via `tmux -C`) | **Stub** | `server/api_tmux.go:305` returns hardcoded `"pane"` |
| Auto-reconnect on disconnect | **Done** | |
| TLS/HTTPS support | **Done** | |
| Touch scroll gesture (mobile) | **Done** | |
| Quick dirs for new windows | **Stub** | `server/api_tmux.go:28` returns `[]` |

---

## 2. AI Assistant Detection & Status

| Feature | Status | Notes |
|---------|--------|-------|
| Detect Claude Code / Codex from command | **Done** | `ai/detector.go` |
| Session title display (first user message) | **Done** | `ai/session_scanner.go` |
| Real-time status tracking (idle/working/approval) | **Missing** | Ref: CodeKanban parses terminal output via vt10x virtual terminal emulator; opencode-tmuxweb uses OpenCode plugin HTTP events |
| Completion notification (working → idle) | **Missing** | Ref: CodeKanban, opencode-tmuxweb |
| Approval-needed notification | **Missing** | Ref: CodeKanban, opencode-tmuxweb |
| Auto-rename session tab from AI input | **Missing** | Ref: CodeKanban |
| Auto-create task when AI starts working | **Missing** | Ref: CodeKanban |
| Qwen Code / Gemini / Cursor detection | **Missing** | Ref: CodeKanban |

### Next Steps for AI Status

`ai/detector.go` handles command-line pattern detection. To add runtime state tracking:

1. Terminal output parsing (or plugin-based event injection)
2. State machine for assistant lifecycle
3. WebSocket metadata broadcast to frontend
4. Notification UI components

---

## 3. AI Session History & Conversation Viewing

| Feature | Status | Notes |
|---------|--------|-------|
| Scan Claude session directories | **Done** | `ai/session_scanner.go` |
| Scan Codex session directories | **Done** | |
| Conversation parsing (messages + tool use) | **Done** | `ai/conversation_parser.go` |
| Conversation viewer UI | **Done** | `frontend/src/conversations/` |
| Database caching of session metadata | **Done** | `service/ai_session.go` |
| Link sessions to Kanban tasks | **Done** | `api_task_ai_session.go` |
| Cleanup stale sessions | **Done** | |
| Phased scanning (recent + background) | **Partial** | Single scan currently. Ref: CodeKanban does 24h immediate + 15d background |
| Cache invalidation (mtime/size check) | **Missing** | Ref: CodeKanban |

---

## 4. AI Command Generation

| Feature | Status | Notes |
|---------|--------|-------|
| OpenAI-compatible API integration | **Done** | `ai/client.go` |
| Built-in AI roles (system prompts) | **Done** | 7 roles |
| Custom AI role CRUD | **Done** | `api_roles.go` |
| Role-specific model/API URL config | **Done** | |
| Send generated command to terminal | **Done** | |
| Command extraction from markdown fences | **Done** | |
| Role manager UI modal | **Done** | `RoleManagerModal.tsx` |

---

## 5. Task Kanban Board

| Feature | Status | Notes |
|---------|--------|-------|
| 4-column board (Todo/In Progress/Done/Archived) | **Done** | |
| Drag-and-drop task movement | **Done** | |
| Task CRUD (title, description, priority, tags, due date) | **Done** | |
| Task comments | **Done** | |
| Task filtering (status, priority, keyword) | **Done** | |
| Task-worktree binding | **Done** | |
| Task-AI session linking | **Done** | |
| Persist to database | **Done** | SQLite |
| Task detail drawer/modal | **Done** | |

---

## 6. Task Segments & Lifecycle Tracking

| Feature | Status | Notes |
|---------|--------|-------|
| Task segment CRUD (per pane) | **Done** | `api_task_segment.go` |
| Chat message recording | **Done** | `db/models.go` |
| Command recording | **Done** | |
| Task summary storage (CRUD) | **Done** | `service/task_segment.go`, `db.TaskSummary` |
| Task lifecycle events API | **Done** | `api_tasks.go` |
| Pane status indicators (sidebar) | **Partial** | Store exists, no real-time AI-driven updates |
| SSE real-time status push | **Partial** | SSE infra exists via Butler proxy + AI streaming; no native `/api/tasks/events/stream` endpoint. Ref: opencode-tmuxweb per-pane event streams |
| Task statistics | **Partial** | `GET /api/panes/status` provides pane-level status; no aggregate task stats endpoint |
| Task summary generation (external AI service) | **Missing** | Storage CRUD done, external AI service call not wired. Ref: opencode-tmuxweb |

---

## 7. Project Management

| Feature | Status | Notes |
|---------|--------|-------|
| Project CRUD | **Done** | `api_project.go` |
| Project-path validation | **Done** | |
| Sync project metadata | **Done** | |
| Priority sorting | **Missing** | |
| File browser / directory listing | **Missing** | Ref: CodeKanban `api/fs.go` |
| Open in external editor (VSCode/Cursor/Zed) | **Missing** | Ref: CodeKanban |

---

## 8. Worktree Management

| Feature | Status | Notes |
|---------|--------|-------|
| Create/list/delete worktrees | **Done** | `service/worktree_service.go` |
| Sync worktree list with repo | **Done** | |
| Commit worktree changes | **Done** | |
| Refresh git status (ahead/behind/modified) | **Done** | |
| go-git native (no exec) | **Done** | |
| Create branch when creating worktree | **Done** | `service/worktree_service.go:140` `createBranch bool` param |

---

## 9. Branch Management

| Feature | Status | Notes |
|---------|--------|-------|
| List branches | **Missing** | Ref: CodeKanban `service/branch_service.go`, worktrunk `wt list` |
| Create branches | **Missing** | Ref: CodeKanban, worktrunk `wt switch -c` |
| Delete branches | **Missing** | Ref: CodeKanban, worktrunk `wt remove` |
| Merge branches (merge/rebase/squash) | **Missing** | Ref: CodeKanban, worktrunk `wt merge` |
| Branch worktree integration | **Missing** | Ref: CodeKanban, worktrunk |

---

## 10. Notepad / Notes

| Feature | Status | Notes |
|---------|--------|-------|
| Multi-tab notes | **Done** | `api_notepad.go` |
| Global notes | **Done** | |
| Project-specific notes | **Done** | |
| Tab reorder | **Done** | |
| Persist to database | **Done** | SQLite |

---

## 11. Imperial Study (Butler Orchestration Dashboard)

| Feature | Status | Notes |
|---------|--------|-------|
| Butler reverse proxy | **Done** | `api_butler.go` |
| SSE passthrough | **Done** | |
| Worker panel | **Done** | |
| Inbox / notifications | **Done** | |
| Activity feed | **Done** | |
| Assistant chat panel | **Done** | |
| Run pipeline view | **Done** | |
| Command input to panes | **Done** | |
| Task detail modal | **Done** | |
| Floating panel mode | **Done** | |
| Transparency slider | **Done** | |

---

## 12. Workspace & Session Organization

| Feature | Status | Notes |
|---------|--------|-------|
| Profile-based workspace switching | **Done** | `api_profiles.go`, MemoryStore |
| Session groups (collapsible) | **Done** | `api_groups.go` |
| Persist profiles/groups to DB | **Missing** | MemoryStore only, data lost on restart. Ref: opencode-tmuxweb (MySQL) |

---

## 13. Voice Input

| Feature | Status | Notes |
|---------|--------|-------|
| Xunfei STT integration | **Done** | `ws_speech.go`, `ai/xunfei.go` |
| WebSocket speech proxy | **Done** | |
| Shake-to-record (mobile) | **Done** | `useShakeDetect.ts` |
| Partial result streaming | **Partial** | |
| Custom hotwords | **Missing** | |

---

## 14. File Upload

| Feature | Status | Notes |
|---------|--------|-------|
| File upload endpoint | **Done** | `api_upload.go`, 10MB limit |
| Copy path to terminal | **Missing** | |
| Date-organized storage | **Missing** | |
| Image preview | **Missing** | |

---

## 15. Mobile Support

| Feature | Status | Notes |
|---------|--------|-------|
| Mobile-optimized UI (`/m` route) | **Done** | |
| Touch toolbox (shortcut keys) | **Done** | |
| Font size slider | **Done** | |
| Keyboard mode toggle | **Done** | |
| iOS keyboard viewport fix | **Done** | `useVisualViewport.ts` |
| iOS burst input suppression | **Partial** | |
| Long-press text selection | **Missing** | |
| Manual reconnect UI | **Done** | `MobileToolbar.tsx`, long-press guarded |

---

## 16. Authentication & Security

| Feature | Status | Notes |
|---------|--------|-------|
| Basic auth | **Done** | Per-request, no session state |
| CSRF protection | **Done** | |
| Rate limiting | **Done** | Per-IP |
| Security headers | **Done** | |
| CORS support | **Done** | |
| User management (multi-user) | **Missing** | |

---

## 17. Persistence & Data

| Feature | Status | Notes |
|---------|--------|-------|
| SQLite database | **Done** | GORM |
| Database auto-migration | **Done** | |
| Profiles/groups/snippets/roles persistence | **Missing** | MemoryStore only |

---

## 18. Configuration

| Feature | Status | Notes |
|---------|--------|-------|
| JSON config file | **Done** | `config/` |
| Environment variable overrides | **Done** | |
| OpenCode config viewer | **Stub** | `server/api_config.go` returns null |
| Theme toggle (light/dark) | **Done** | `hooks/useTheme.ts`, `data-theme` attribute |
| i18n (English/Chinese) | **Done** | `i18n/` with i18next, en/zh translations |
| Hot-reload config | **Missing** | |
| Version endpoint | **Missing** | `middleware.go:21` TODO |
| Update checker | **Missing** | |

---

## 19. Developer Experience

| Feature | Status | Notes |
|---------|--------|-------|
| Server log endpoint | **Done** | `api_log.go` |
| Telemetry endpoint | **Done** | `api_telemetry.go` |
| OpenAPI docs | **Missing** | |
| API capabilities endpoint | **Missing** | |

---

## Reference: worktrunk CLI Patterns

worktrunk is a Rust CLI for git worktree management. These patterns are worth studying for web UI equivalents.

| Feature | Reference |
|---------|-----------|
| Interactive worktree picker (live diff/log preview) | worktrunk `wt switch` interactive mode |
| Hooks system (create, pre-merge, post-merge, etc.) | worktrunk hooks |
| LLM commit messages (generate from diffs) | worktrunk `wt step commit` |
| Build cache sharing (copy `target/`, `node_modules/`) | worktrunk `wt step` |
| PR checkout (`wt switch pr:123`) | worktrunk PR integration |
| Agent launch (`wt switch -x claude`) | worktrunk agent integration |
| CI status per branch | worktrunk `wt list --full` |
| LLM-generated branch summaries | worktrunk `wt list --full` |
| Configurable worktree path template | worktrunk path templates |

---

## What Needs Work

### Stubs to Implement (3)

| Stub | File | Effort |
|------|------|--------|
| Quick dirs | `server/api_tmux.go:28` | Low — read from config |
| Pane mode toggle | `server/api_tmux.go:295` | Medium — tmux control mode support |
| OpenCode config | `server/api_config.go` | Medium — read opencode.json from pane cwd |

### Missing Features (High Priority)

| Feature | Effort | Reference |
|---------|--------|-----------|
| AI assistant real-time status tracking | High | CodeKanban (vt10x parsing), opencode-tmuxweb (plugin events) |
| Branch management (list/create/merge/delete) | Medium | CodeKanban `service/branch_service.go`, worktrunk |
| Merge workflow (squash/rebase/merge + cleanup) | Medium | CodeKanban, worktrunk `wt merge` |
| Persist profiles/groups/snippets to DB | Low | opencode-tmuxweb (MySQL) |
| Native SSE task events endpoint | Medium | opencode-tmuxweb (per-pane event streams) |
| Task summary generation (external AI service call) | Medium | opencode-tmuxweb |
| Cache invalidation for AI sessions | Low | CodeKanban |
| File browser / directory listing | Low | CodeKanban `api/fs.go` |
| Quick dirs (configurable) | Low | opencode-tmuxweb |

### Missing Features (Lower Priority)

| Feature | Effort | Reference |
|---------|--------|-----------|
| i18n support | High | CodeKanban, opencode-tmuxweb |
| Advanced mobile gesture handling | Medium | opencode-tmuxweb |
| Open in external editor | Low | CodeKanban |
| Aggregate task statistics endpoint | Low | opencode-tmuxweb |
| Version / update checker | Low | CodeKanban |
| Hot-reload config | Medium | CodeKanban |
| OpenAPI docs generation | Medium | CodeKanban (Huma framework) |
| LLM commit messages | Medium | worktrunk |
| PR checkout (create worktree from PR) | Medium | worktrunk |
| Worktree hooks (lifecycle automation) | Medium | worktrunk |
| Build cache sharing between worktrees | Low | worktrunk |
