# CLAUDE.md — db/

SQLite database layer with GORM ORM. CGO-free via `glebarez/sqlite`. Thread-safe singleton with WAL mode.

## File Layout

```
db.go          Init(), GetDB(), IsAvailable(), Close() — singleton lifecycle
models.go      GORM models: TaskSegment, ChatMessage, CommandRecord, TaskSummary, TaskAISession
migrate.go     Auto-migration registration
options.go     DefaultOptions() — default DB path (~/.local/share/ttyweb/ttyweb.db)
errors.go      ErrNotInitialized and sentinel errors
task_ai_session.go  Many-to-many task↔AI session linking table
```

## Key Patterns

**Singleton**: `globalDB` protected by `sync.Mutex`. `Init()` is idempotent — safe to call multiple times. `Close()` nils the global only after successful close.

**WAL Mode**: Enabled on init for better concurrent read performance.

**Graceful Degradation**: If DB init fails, `IsAvailable()` returns false. Server continues operating with in-memory-only data. Handlers check `db.IsAvailable()` before DB-dependent operations.

**Time Partitioning**: Models include `Year` and `Mon` fields with composite indexes for efficient range queries. Partitioning is application-level (not SQLite-native).

**Auto-Migration**: Models are registered via `migrate.go` and applied on `Init()`. New models just need adding to the registration list.

## Models

| Model | Purpose | Key Indexes |
|-------|---------|-------------|
| `TaskSegment` | AI task lifecycle in a pane | session, pane, year/mon |
| `ChatMessage` | Conversation messages per segment | segment_id, year/mon |
| `CommandRecord` | Terminal commands per segment | segment_id, year/mon |
| `TaskSummary` | Generated summaries per segment | segment_id, session/window |
| `TaskAISession` | Many-to-many task↔AI session link | — |

## Testing

```bash
go test ./db/... -v -count=1
go test ./db/... -run TestInit -v
go test ./db/... -run TestOptions -v
```

Tests use in-memory SQLite (`:memory:`). No external dependencies.
