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
| Tmux control mode (single PTY via `tmux -C`) | **Stub** | Returns hardcoded `"pane"` mode |
| Auto-reconnect on disconnect | **Done** | |
| TLS/HTTPS support | **Done** | |
| Touch scroll gesture (mobile) | **Done** | |
| Quick dirs for new windows | **Stub** | Returns empty array |

---

## 2. AI Assistant Detection & Status

| Feature | Status | Notes |
|---------|--------|-------|
| Detect Claude Code / Codex from command | **Done** | `ai/detector.go` |
| Session title display (first user message) | **Done** | `ai/session_scanner.go` |
| Real-time status tracking (idle/working/approval) | **Done** | `ai/state_machine.go`, `ai/interceptor.go` |
| Completion notification (working → idle) | **Done** | `NotificationProvider.tsx`, toast notifications |
| Approval-needed notification | **Done** | `NotificationProvider.tsx`, toast notifications |
| Auto-rename session tab from AI input | **Done** | `handlers.go`, detects AI user messages |
| Auto-create task when AI starts working | **Done** | `handlers.go`, detects AI working state |
| Qwen Code / Gemini / Cursor detection | **Done** | `ai/detector.go` |

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
| Phased scanning (recent + background) | **Done** | `service/ai_session.go`, recent immediate + background refresh |
| Cache invalidation (mtime/size check) | **Done** | `service/ai_session.go`, mtime/size validation on scan |

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
| Pane status indicators (sidebar) | **Done** | Real-time AI-driven updates via state machine |
| SSE real-time status push | **Done** | `server/event_bus.go`, `TaskEventBus` pub/sub with SSE streaming |
| Task statistics | **Done** | `service/stats_service.go`, `GET /api/tasks/stats` with daily breakdown |
| Task summary generation (external AI service) | **Done** | `service/summary_service.go`, `POST /api/segments/{id}/summarize` via LLM |

---

## 7. Project Management

| Feature | Status | Notes |
|---------|--------|-------|
| Project CRUD | **Done** | `api_project.go` |
| Project-path validation | **Done** | |
| Sync project metadata | **Done** | |
| Priority sorting | **Missing** | |
| File browser / directory listing | **Done** | `server/api_fs.go`, `frontend/src/filebrowser/` |
| Open in external editor (VSCode/Cursor/Zed) | **Done** | `server/api_editor.go`, generates `vscode://` / `cursor://` URLs |

---

## 8. Worktree Management

| Feature | Status | Notes |
|---------|--------|-------|
| Create/list/delete worktrees | **Done** | `service/worktree_service.go` |
| Sync worktree list with repo | **Done** | |
| Commit worktree changes | **Done** | |
| Refresh git status (ahead/behind/modified) | **Done** | |
| go-git native (no exec) | **Done** | |
| Create branch when creating worktree | **Done** | `service/worktree_service.go` `createBranch bool` param |
| Worktree hooks (lifecycle automation) | **Done** | `worktree/hooks.go`, pre-create/post-create/pre-merge/post-merge/pre-remove |
| PR checkout (create worktree from PR) | **Done** | `worktree/pr.go`, `service/pr_checkout.go` |
| LLM commit messages (generate from diffs) | **Done** | `service/commit_message.go`, `POST .../ai-commit-message` |
| Worktree diff generation | **Done** | `worktree/worktree_diff.go`, unified diff output |

---

## 9. Branch Management

| Feature | Status | Notes |
|---------|--------|-------|
| List branches | **Done** | `worktree/branch.go`, `GET /api/branches?repo=<path>` |
| Create branches | **Done** | `worktree/branch.go`, `POST /api/branches` |
| Delete branches | **Done** | `worktree/branch.go`, `DELETE /api/branches/{name}` |
| Merge branches (merge/rebase/squash) | **Done** | `worktree/branch.go`, `POST /api/branches/merge` |
| Branch worktree integration | **Done** | `worktree/branch.go`, ahead/behind tracking per branch |

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
| Persist profiles/groups to DB | **Done** | `service/persist_service.go`, GORM persistence |

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
| Profiles/groups/snippets/roles persistence | **Done** | `service/persist_service.go` |

---

## 18. Configuration

| Feature | Status | Notes |
|---------|--------|-------|
| JSON config file | **Done** | `config/` |
| Environment variable overrides | **Done** | |
| OpenCode config viewer | **Stub** | `server/api_config.go` returns null |
| Theme toggle (light/dark) | **Done** | `hooks/useTheme.ts`, `data-theme` attribute |
| i18n (English/Chinese) | **Done** | `i18n/` with i18next, en/zh translations |
| Hot-reload config | **Done** | `config/watch.go`, fsnotify-based |
| Version endpoint | **Done** | `server/version.go`, `GET /version` |
| Update checker | **Done** | `server/update_checker.go`, GitHub release API |

---

## 19. Developer Experience

| Feature | Status | Notes |
|---------|--------|-------|
| Server log endpoint | **Done** | `api_log.go` |
| Telemetry endpoint | **Done** | `api_telemetry.go` |
| OpenAPI docs | **Done** | `server/api_swagger.go`, Swagger UI at `/api/docs` |
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

### Stubs to Implement (0)

All previous stubs have been implemented.

### Missing Features (Lower Priority)

| Feature | Effort | Reference |
|---------|--------|-----------|
| Priority sorting for projects | Low | — |
| API capabilities endpoint | Low | — |
| File upload enhancements (copy path, date-organized, image preview) | Medium | CodeKanban |
| User management (multi-user) | High | — |
| Custom hotwords for voice input | Medium | — |
| Long-press text selection (mobile) | Medium | opencode-tmuxweb |
| iOS burst input suppression | Medium | opencode-tmuxweb |
| Build cache sharing between worktrees | Low | worktrunk |
| Configurable worktree path template | Low | worktrunk |
| CI status per branch | Low | worktrunk |
