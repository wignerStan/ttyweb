# CLAUDE.md — config/

JSON configuration system with environment variable overrides.

## File Layout

```
config.go    Load(), Get(), LoadOrDefault(), applyEnvOverrides()
defaults.go  DefaultConfig(), DefaultConfigPath()
watch.go     Reload(), Watch() — hot-reload via fsnotify
```

## Configuration Sections

| Section | JSON Key | Env Overrides |
|---------|----------|---------------|
| LLM | `llm` | `LLM_API_KEY`, `LLM_API_URL`, `LLM_MODEL` |
| Xunfei STT | `xfyun` | `XFYUN_APP_ID`, `XFYUN_API_KEY`, `XFYUN_API_SECRET` |
| Butler proxy | `butler` | `BUTLER_HOST`, `BUTLER_PORT` |
| Database | `db` | — |
| Worktree | `worktree` | — |

## Key Patterns

**Config Path**: `~/.config/ttyweb/config.json` (respects `XDG_CONFIG_HOME`). Overridable via `-config` CLI flag.

**Env Vars Win**: Non-empty environment variables always override JSON file values.

**Singleton Cache**: `LoadOrDefault()` caches the result via `sync.Once`. `Get()` returns a shallow copy (safe to mutate).

**Missing File OK**: `Load()` returns defaults if config file doesn't exist. Only fails on parse errors.

**Defaults**: OpenAI API (`https://api.openai.com/v1/chat/completions`), `gpt-4o` model, Butler at `localhost:8215`.

**Hot Reload**: `Reload(path)` re-reads the config file and swaps `globalConfig`. `Watch(path)` monitors the file via fsnotify and publishes updates to a channel. Caller owns the stop function.

## Testing

```bash
go test ./config/... -v -count=1
```
