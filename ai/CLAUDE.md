# CLAUDE.md — ai/

AI/LLM integration: OpenAI-compatible chat client, Claude Code/Codex session detection, conversation parsing, streaming WebSocket, Xunfei STT.

## File Layout

```
client.go              OpenAI-compatible chat completion (HTTP POST, 60s timeout)
stream.go              Streaming LLM response via WebSocket
roles.go               7 built-in AI role system prompts
detector.go            Detect Claude Code / Codex from command-line patterns
session_scanner.go     Scan Claude/Codex session directories for JSONL files
conversation_parser.go Parse JSONL conversation files (messages, tool use blocks)
xunfei.go              Xunfei STT WebSocket proxy (HMAC-SHA256 auth)
log_watcher.go         Watch AI agent log files for new activity
types.go               Shared types (Session, ParsedMessage, ToolUse, etc.)
```

## Key Patterns

**OpenAI Client**: Stateless HTTP client using `config.Get().LLM` for API key/URL/model. No SDK dependency — raw JSON over HTTP. Returns extracted markdown content from response choices.

**Session Detection**: `detector.go` identifies Claude Code and Codex sessions from process command patterns (e.g., `/claude`, `codex`). Used by `session_scanner.go` to find and parse agent session directories.

**Conversation Parsing**: JSONL files are parsed into `ParsedMessage` structs with `ToolUse` blocks. Supports grouping by conversation segments.

**Streaming**: `stream.go` pipes LLM responses to WebSocket clients with role-based system prompts.

**Xunfei STT**: WebSocket proxy with HMAC-SHA256 authentication. Credentials from `config.Get().Xunfei`.

## Testing

```bash
go test ./ai/... -v -count=1
go test ./ai/... -run TestClient -v
go test ./ai/... -run TestDetector -v
go test ./ai/... -run TestConversationParser -v
go test ./ai/... -run TestSessionScanner -v
```
