# CLAUDE.md — webtty/

Core protocol layer bridging `backend.Slave` to WebSocket connections.

## Binary Protocol

Single-byte type prefix + base64 payload, negotiated via WebSocket subprotocol `"webtty"`.

**Client → Server:**
| Byte | Name | Description |
|------|------|-------------|
| `0` | UnknownInput | Bug/unexpected |
| `1` | Input | Keyboard input |
| `2` | Ping | Keepalive |
| `3` | ResizeTerminal | Terminal dimensions changed |
| `4` | SetEncoding | Change character encoding |

**Server → Client:**
| Byte | Name | Description |
|------|------|-------------|
| `0` | UnknownOutput | Bug/unexpected |
| `1` | Output | Terminal output |
| `2` | Pong | Keepalive response |
| `3` | SetWindowTitle | Update browser tab title |
| `4` | SetPreferences | Terminal preferences |
| `5` | SetReconnect | Trigger client reconnection |
| `6` | SetBufferSize | Configure input buffer size |

Defined in `message_types.go`.

## Configuration Options

Options are applied via the functional options pattern when creating a new WebTTY instance.

| Option | Purpose | Default |
|--------|---------|---------|
| `WithPermitWrite()` | Allow slave to accept input from clients | false |
| `WithFixedColumns(int)` | Set fixed terminal width | 0 (dynamic) |
| `WithFixedRows(int)` | Set fixed terminal height | 0 (dynamic) |
| `WithTitleFormat([]byte)` | Set default window title | nil |
| `WithReconnect(int)` | Enable client reconnection (seconds) | 0 (disabled) |
| `WithMasterPreferences(any)` | Set master preferences (JSON) | nil |

Defined in `option.go`.

## File Layout

```
webtty.go        Main WebTTY struct (Slave ↔ Master bridge)
master.go        WebSocket-side interface (reads client messages, writes server messages)
slave.go         Backend-side interface (reads slave output, writes slave input)
codecs.go        Base64 encoding/decoding for the protocol
option.go        Configuration options
errors.go        Error types
```

## Testing

```bash
go test ./webtty/... -v -count=1
go test ./webtty/... -run TestWebTTY -v
go test ./webtty/... -run TestCodec -v
```
