# CLAUDE.md — internal/

Internal utilities not intended for external import.

## File Layout

```
slogutil/           Structured JSON logging factory
  slogutil.go       New(w, level) creates slog.Logger with configurable level
```

## Key Patterns

**Structured Logging**: `slogutil.New(level)` creates a `*slog.Logger` writing JSON to stdout. Used throughout the codebase after the migration from `log.Printf`. Log level is configurable (debug/info/warn/error).

## Testing

```bash
go test ./internal/... -v -count=1
go test ./internal/slogutil/... -v
```
