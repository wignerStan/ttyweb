# CLAUDE.md — frontend/

React SPA (Vite + TypeScript) providing terminal UI with desktop and mobile layouts. Uses **bun** as package manager.

## Build & Dev

```bash
bun install          # Install dependencies
bun run build        # Build to ../bindata/static/ (Go embed target)
bun run dev          # Dev server on :5173, proxies /api and /ws to :8080
```

## Testing

```bash
bunx vitest run                    # Unit tests
bunx vitest run -t "renders"       # Single test by name
bun run typecheck                  # Type checking (tsc --noEmit)
bunx playwright test               # E2E (all backends)
bunx playwright test --project=local  # E2E (single backend)
bunx playwright test -g "auth"     # E2E (by test name)
```

## Linting & Formatting

```bash
bun run lint           # Biome check src/
bun run lint:fix       # Biome auto-fix
bun run format         # Biome format --write src/
bun run format:check   # Biome format check
```

## Architecture

```
src/
├── App.tsx            Desktop layout: Routes / (desktop) → /m (mobile)
├── types.ts           TypeScript interfaces matching Go API types
├── components/
│   ├── Sidebar.tsx    Session/pane tree navigation
│   └── TerminalTab.tsx xterm.js terminal (FitAddon, WebLinksAddon)
├── hooks/
│   ├── useNewWindow.ts     Open terminal in new browser window
│   ├── useAIConversations.ts  AI session/conversation data
│   ├── useTmuxPrefix.ts    Tmux prefix key configuration
│   ├── useShakeDetect.ts   Shake-to-record on mobile
│   ├── useKeyboardAvoider.ts  Mobile keyboard avoidance
│   └── useVisualViewport.ts  Viewport handling
├── conversations/       AI conversation history viewer
├── kanban/              Task board with drag-and-drop (@dnd-kit)
├── notepad/             Multi-tab notepad
├── mobile/
│   ├── MobileApp.tsx      Mobile route handler
│   ├── MobileTerminal.tsx Mobile xterm.js instance
│   ├── MobileToolbar.tsx  Tab bar + actions
│   ├── MobileDrawer.tsx   Slide-out navigation
│   └── MobileToolbox.tsx  Keyboard toolbox overlay
├── shared/components/     Shared UI components
├── utils/
│   ├── auth.ts            Basic auth handling
│   ├── platform.ts        Platform detection
│   ├── rlog.ts            Remote logging
│   └── telemetry.ts       Telemetry emitter
├── index.css              Global styles (Tokyo Night theme)
├── main.tsx               React entry point
└── test-setup.ts          Vitest setup
```

## Key Patterns

**Frontend State**: No global state library. Component-local state with React hooks. Mobile app persists tabs to localStorage.

**Terminal Rendering**: `TerminalTab` wraps xterm.js with `FitAddon` (auto-resize to container) and `WebLinksAddon` (clickable URLs). Each tab instance gets its own WebSocket and terminal.

**WebSocket Protocol**: Base64-encoded messages matching the WebTTY protocol (see `webtty/CLAUDE.md`). Auto-reconnect handled at the connection level.

**Styling**: Tokyo Night color scheme hardcoded in inline styles (no CSS framework). Monospace font stack: JetBrains Mono → Fira Code → Cascadia Code → monospace.

**Mobile Route** (`/m`): Separate optimized layout with tab management, font size slider, voice input, shake detection, and keyboard toolbox.

**Linting**: Biome (not ESLint) for linting and formatting. Config in `biome.json`.

## E2E Tests

Playwright tests in `e2e/` run against three backend projects configured in `playwright.config.ts`:
- `local` (port 18899) — no session management
- `tmux` (port 18900) — full session+pane management
- `zellij` (port 18901) — session management only (no panes)

All 3 servers start via top-level `webServer` array. `reuseExistingServer` is enabled outside CI.

### Test Files

| File | Coverage |
|------|----------|
| `websocket.spec.ts` | WebSocket connection lifecycle |
| `terminal-io.spec.ts` | Terminal input/output |
| `api.spec.ts` | REST API endpoints |
| `api-edge-cases.spec.ts` | Error handling, edge cases |
| `auth.spec.ts` | Authentication flows |
| `session-ui.spec.ts` | Session management UI |
| `frontend.spec.ts` | General UI rendering |
| `mobile.spec.ts` | Mobile-specific UI |

### Helpers (`e2e/helpers.ts`)

- `hasSessionManagement(request)` — checks `/api/backends`, true for tmux/zellij only
- `hasSessionsWithPanes(page)` — checks tree API, false for zellij (no panes)
- Session CRUD helpers, sidebar DOM polling, WebSocket monitoring, mobile helpers

Use `test.skip(!await hasSessionManagement(request), '...')` to skip session-dependent tests on local.

### Fixtures (`e2e/fixtures.ts`)

Composes `@playwright/test` with `@seontechnologies/playwright-utils`:
- `apiRequest` — auto-retries 5xx
- `networkErrorMonitor` — auto-fails on 4xx/5xx (causes false positives on zellij where many endpoints return 503)
