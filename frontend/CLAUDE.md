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
│   ├── useVisualViewport.ts  Viewport handling
│   ├── useTheme.ts              Theme management with localStorage persistence
│   └── useLongPress.ts          Long-press gesture detection for touch
├── conversations/       AI conversation history viewer
├── kanban/              Task board with drag-and-drop (@dnd-kit)
├── notepad/             Multi-tab notepad
├── branches/            Branch list, create, delete operations
├── filebrowser/         Directory listing with breadcrumb navigation
├── i18n/                English and Chinese translations (i18next)
├── mobile/
│   ├── MobileApp.tsx      Mobile route handler
│   ├── MobileTerminal.tsx Mobile xterm.js instance
│   ├── MobileToolbar.tsx  Tab bar + actions
│   ├── MobileDrawer.tsx   Slide-out navigation
│   └── MobileToolbox.tsx  Keyboard toolbox overlay
├── shared/components/     Shared UI components
│   ├── NotificationProvider.tsx  Toast notification system (AI events, errors)
│   ├── ThemeToggle.tsx           Dark/light theme toggle
│   └── styles/theme.css          Theme CSS variables
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

**Styling**: Tailwind CSS v4 + DaisyUI v5. Tokyo Night color scheme via DaisyUI custom themes ("dark" default, "light"). Semantic design tokens in `app.css` @theme block. DaisyUI component classes (btn, input, select, modal, badge, etc.) used across UI. Monospace font stack: JetBrains Mono → Fira Code → Cascadia Code → monospace. Focus ring system, reduced-motion support, and 44px touch targets on mobile.

**Mobile Route** (`/m`): Separate optimized layout with tab management, font size slider, voice input, shake detection, and keyboard toolbox.

**Linting**: Biome (not ESLint) for linting and formatting. Config in `biome.json`.

**Branch Management**: `branches/BranchPanel.tsx` provides branch list, create, delete operations with current branch highlighting and ahead/behind status. Data from `GET /api/branches`.

**File Browser**: `filebrowser/FileBrowser.tsx` provides directory listings with breadcrumb navigation, file sizes, and modification times. Data from `GET /api/fs`.

**i18n**: `i18n/` uses i18next with English and Chinese translations. Language preference persisted to localStorage. Import `i18n/index.ts` in `main.tsx`.

**Theme System**: `useTheme.ts` manages dark/light mode via `data-theme` attribute on `<html>`. Persisted to localStorage. Theme CSS variables in `styles/theme.css`.

**Notifications**: `NotificationProvider.tsx` provides toast notifications for AI state changes (completion, approval-needed), errors, and general info. Auto-dismiss with type-based styling.

**Long Press**: `useLongPress.ts` detects long-press gestures on touch devices for context menus.

## E2E Tests

Playwright tests in `e2e/` run against three backend projects configured in `playwright.config.ts`:
- `local` (port 18899) — no session management
- `tmux` (port 18900) — full session+pane management
- `zellij` (port 18901) — session management only (no panes)

All 3 servers start via top-level `webServer` array. `reuseExistingServer` is enabled outside CI.

### Playwright Config

Key settings in `playwright.config.ts`:
- `fullyParallel: true` — tests run in parallel by default
- `retries: process.env.CI ? 2 : 0` — no local retries (expose flakiness), 2 retries in CI
- `workers: process.env.CI ? 2 : undefined` — auto-detect locally, 2 in CI
- `navigationTimeout: 15_000` — faster than default 30s
- `video: 'retain-on-failure'` in CI, `'off'` locally
- `trace: 'on-first-retry'` — only captured on retry

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
| `mobile.spec.ts` | Mobile-specific UI (requires sessions with panes) |
| `config-swagger.spec.ts` | Config, auth, pane status, telemetry, health, upload, AI, swagger, log |
| `theme-i18n-mobile.spec.ts` | Theme toggle, i18n, mobile layout, responsive, accessibility, performance |
| `branch-api.spec.ts` | Branch and file browser endpoints |
| `persistence-events.spec.ts` | Profiles, groups, snippets, tasks CRUD lifecycle, SSE, roles |
| `stress-concurrency.spec.ts` | Rapid sequential, concurrent, large payloads, boundary values, rate limiting |
| `misc-api.spec.ts` | Miscellaneous API endpoints (profiles, snippets, roles, telemetry, etc.) |

### Helpers (`e2e/helpers.ts`)

- `hasSessionManagement(request)` — checks `/api/backends`, true for tmux/zellij only
- `hasSessionsWithPanes(page)` — checks tree API, false for zellij (no panes)
- Session CRUD helpers, sidebar DOM polling, WebSocket monitoring, mobile helpers

Use `test.skip(!await hasSessionManagement(request), '...')` to skip session-dependent tests on local.

### Fixtures (`e2e/fixtures.ts`)

Composes `@playwright/test` with `@seontechnologies/playwright-utils/api-request`:
- `apiRequest` — auto-retries 5xx

Note: `networkErrorMonitor` was removed — it auto-failed on 4xx/5xx, conflicting with error-testing specs.

### Locator Conventions

Follow Playwright locator priority: **Role > Label > Text > TestID > CSS/XPath**.

- Use `page.getByRole('heading', { name: 'Sessions' })` not `page.locator('h3:has-text("Sessions")')`
- Use `page.getByPlaceholder('new session')` not `page.locator('input[placeholder="new session"]')`
- Use `page.getByText('connected')` not `page.locator('text=connected')`
- Use `page.getByTitle('Kill session')` for titled buttons
- Prefer `.click()` over `.dispatchEvent('click')` — regular clicks benefit from Playwright's auto-waiting and actionability checks
- Avoid `{ force: true }` — it bypasses actionability checks and may hide real issues
