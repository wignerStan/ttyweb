# CLAUDE.md — service/

Business logic layer between API handlers and data stores. Orchestrates `db`, `ai`, and `worktree` packages.

## File Layout

```
ai_session.go          AI session CRUD, scanning, database caching, cleanup
notepad.go             Multi-tab notepad (global + project-specific), auto-save
project.go             Project CRUD with git directory validation
task_segment.go        Task lifecycle: segments, chat messages, command records
task_ai_session.go     Many-to-many task↔AI session linking
worktree_service.go    Worktree operations with project binding
persist_service.go     Database persistence for profiles, groups, and snippets (replaces MemoryStore)
commit_message.go      AI-powered conventional commit message generation from git diffs
summary_service.go     AI-powered task summary generation using commands + chat context
stats_service.go       Aggregate task statistics (counts, daily completions, status breakdown)
pr_checkout.go         PR checkout orchestration (fetch PR, validate, create worktree)
cache.go               Generic SharedCache with TTL expiration
helpers.go             Shared utility functions
```

## Key Patterns

**Service Layer**: Server handlers (`server/api_*.go`) call service functions rather than accessing `db` or `ai` directly. This decouples HTTP concerns from business logic.

**AI Session Store**: `ai_session.go` maintains an in-memory cache of scanned AI sessions with periodic refresh. Falls back to fresh scan when cache is stale.

**Notepad**: Supports multiple tabs with reorder, project association, and auto-save. Global notepads (no project) and project-scoped notepads.

**Task Tracking**: `task_segment.go` manages the full task lifecycle — segments represent AI tasks within terminal panes, linked to chat messages and command records.

**Project Validation**: `project.go` validates that project paths contain valid git repositories before creation.

**Persistence Service**: `persist_service.go` provides GORM-backed CRUD for profiles, groups, and snippets. Replaces in-memory MemoryStore for these entities. Used by `server/api_profiles.go`, `api_groups.go`, `api_snippets.go`.

**AI Commit Messages**: `commit_message.go` generates conventional commit messages from git diffs using the configured LLM. Returns type, scope, subject, and body.

**Task Summaries**: `summary_service.go` generates AI summaries for task segments using command records and chat messages as context. Stored in `db.TaskSummary`.

**Task Statistics**: `stats_service.go` provides aggregate counts, daily completion breakdowns, and status distribution from task segment data.

**PR Checkout**: `pr_checkout.go` orchestrates the full PR checkout workflow — fetches PR details from GitHub, validates repo state, creates local worktree branch.

**SharedCache**: `cache.go` is a generic TTL-based cache used by AI session scanning and other services. Thread-safe with automatic cleanup.

## Testing

```bash
go test ./service/... -v -count=1
go test ./service/... -run TestAISession -v
go test ./service/... -run TestNotepad -v
go test ./service/... -run TestProject -v
go test ./service/... -run TestTaskSegment -v
go test ./service/... -run TestWorktreeService -v
```
