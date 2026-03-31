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
helpers.go             Shared utility functions
```

## Key Patterns

**Service Layer**: Server handlers (`server/api_*.go`) call service functions rather than accessing `db` or `ai` directly. This decouples HTTP concerns from business logic.

**AI Session Store**: `ai_session.go` maintains an in-memory cache of scanned AI sessions with periodic refresh. Falls back to fresh scan when cache is stale.

**Notepad**: Supports multiple tabs with reorder, project association, and auto-save. Global notepads (no project) and project-scoped notepads.

**Task Tracking**: `task_segment.go` manages the full task lifecycle — segments represent AI tasks within terminal panes, linked to chat messages and command records.

**Project Validation**: `project.go` validates that project paths contain valid git repositories before creation.

## Testing

```bash
go test ./service/... -v -count=1
go test ./service/... -run TestAISession -v
go test ./service/... -run TestNotepad -v
go test ./service/... -run TestProject -v
go test ./service/... -run TestTaskSegment -v
go test ./service/... -run TestWorktreeService -v
```
