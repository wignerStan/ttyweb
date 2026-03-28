# TEA Assessment Report — ttyweb

> Generated: 2026-03-28
> Methodology: TEA (Test Engineering Architect) — Integrated Brownfield
> Scope: Full codebase (Go backend, React frontend, E2E, security)

## Executive Summary

| Metric | Value |
|--------|-------|
| **Codebase** | 2,831 lines Go + 677 lines React/TypeScript |
| **Go test files** | 2 / 33 source files (~20-30% coverage) |
| **Frontend unit tests** | 21 tests across 3 files |
| **E2E tests** | 31 Playwright + ~16 CLI script |
| **Security findings** | 3 CRITICAL, 6 HIGH, 7 MEDIUM, 5 LOW |
| **TEA quality score** | Deterministic 4/10, Isolated 3/10, Explicit 7/10, Focused 6/10, Fast 6/10 |
| **CI/CD** | None |
| **Recommended model** | TEA Integrated (Brownfield) |

**Verdict:** The architecture is clean and well-designed, but test coverage is thin for a web terminal that exposes shell access. The security posture has critical gaps that should be addressed before any production deployment.

---

## 1. Coverage Heatmap

### Go Backend

| Package | Test Files | Coverage | Target | Gap | Risk |
|---------|-----------|----------|--------|-----|------|
| `webtty/webtty` | 1 (6 tests) | ~60% | 100% | 40% | P0 |
| `server/` | 0 | 0% | 90% | 90% | P0/P1 |
| `backend/tmux/` | 0 | 0% | 90% | 90% | P1 |
| `backend/zellij/` | 0 | 0% | 90% | 90% | P1 |
| `backend/localcommand/` | 1 (2 tests) | ~40% | 90% | 50% | P1 |
| `pkg/` | 0 | 0% | 50% | 50% | P2 |

### Frontend (React/TypeScript)

| Component | Unit Tests | Coverage | Target | Gap | Risk |
|-----------|-----------|----------|--------|-----|------|
| `TerminalTab.tsx` | 7 | ~25% | 100% | 75% | P0 |
| `Sidebar.tsx` | 8 | ~45% | 90% | 45% | P1 |
| `App.tsx` | 6 | ~55% | 50% | Met | P2 |

### E2E

| System | Tests | Status |
|--------|-------|--------|
| Playwright specs (`frontend/e2e/`) | 31 | Active, well-structured |
| Shell E2E (`test_e2e.sh`) | ~20 | Legacy, race conditions |
| CLI wrapper (`playwright-cli-tests.sh`) | ~16 | Legacy, overlapping |

---

## 2. Risk Register

TEA scoring: Probability (1-3) x Impact (1-3) = Risk Score (1-9)

| # | Risk | Category | P | I | Score | Gate Decision |
|---|------|----------|---|---|-------|---------------|
| R1 | Command injection via `PermitArguments` | SEC | 3 | 3 | **9** | BLOCKS RELEASE |
| R2 | Command injection via tmux/zellij session names | SEC | 3 | 3 | **9** | BLOCKS RELEASE |
| R3 | REST API exposed without auth by default | SEC | 3 | 3 | **9** | BLOCKS RELEASE |
| R4 | `PermitWrite` defaults true + no auth | SEC | 3 | 3 | **9** | BLOCKS RELEASE |
| R5 | WS origin check defaults allow-all (CSWSH) | SEC | 2 | 3 | **6** | CONCERNS |
| R6 | WebSocket protocol msg handling untested (P0) | TECH | 3 | 3 | **9** | BLOCKS RELEASE |
| R7 | Entire `server/` package has zero tests | TECH | 2 | 3 | **6** | CONCERNS |
| R8 | No CI pipeline | OPS | 2 | 2 | 4 | Mitigate |
| R9 | 3 overlapping E2E systems, race conditions | OPS | 2 | 2 | 4 | Mitigate |
| R10 | Cleartext credentials over WS, timing attack | SEC | 2 | 2 | 4 | Mitigate |
| R11 | `homedir.Expand()` panics on short strings | BUG | 2 | 2 | 4 | Mitigate |
| R12 | No rate limiting on auth endpoints | SEC | 2 | 2 | 4 | Mitigate |
| R13 | Unbounded memory read in WS wrapper | SEC | 2 | 2 | 4 | Mitigate |
| R14 | No CSRF protection on REST API | SEC | 2 | 2 | 4 | Mitigate |
| R15 | TLS config missing security hardening | SEC | 1 | 2 | 2 | Monitor |
| R16 | Error messages leak internal info | SEC | 1 | 2 | 2 | Monitor |

---

## 3. Security Findings

### CRITICAL (3)

#### S1. Command Injection via `PermitArguments` + Client Query Parameters
- **File:** `server/handlers.go:112-121`
- **Impact:** Client-supplied URL query params passed directly as command-line arguments
- **Fix:** Whitelist allowed arguments; default `PermitArguments` to `false`

#### S2. Command Injection via tmux/zellij Session and Pane Names
- **Files:** `backend/tmux/tmux.go:48-59`, `backend/tmux/sessions.go` (multiple), `backend/zellij/sessions.go` (multiple), `server/api.go:63-65`
- **Impact:** Unsantized session/pane names passed to tmux/zellij CLI
- **Fix:** Validate names against `^[a-zA-Z0-9_.-]+$` before all command invocations

#### S3. REST API Lacks Authentication by Default
- **File:** `server/api.go` (all endpoints)
- **Impact:** `POST /api/sessions`, `DELETE /api/sessions/{name}` reachable without auth
- **Fix:** Require auth for destructive operations; separate API token mechanism

### HIGH (6)

| # | Finding | File | Fix |
|---|---------|------|-----|
| H1 | WS origin check defaults allow-all | `server/server.go:60-79` | Default to restrictive `CheckOrigin` |
| H2 | Credentials in cleartext over WS, timing attack | `server/handlers.go:103-110` | `subtle.ConstantTimeCompare`, warn on no-TLS |
| H3 | No rate limiting on auth | `server/handlers.go`, `server/api.go` | Per-IP rate limiting, exponential backoff |
| H4 | Unbounded memory read in WS wrapper | `server/ws_wrapper.go:34` | Use `io.LimitReader` before processing |
| H5 | `PermitWrite` defaults true | `main.go:42,68` | Default to `false` |
| H6 | No CSRF protection | `server/middleware.go`, `server/api.go` | CSRF tokens or `X-Requested-With` header |

### MEDIUM (7)

| # | Finding | File |
|---|---------|------|
| M1 | TLS config missing MinVersion, cipher suites, timeouts | `server/server.go:220-234` |
| M2 | Missing security headers (CSP, X-Frame-Options, HSTS) | `server/middleware.go:18-23` |
| M3 | `homedir.Expand()` panics on short strings | `pkg/homedir/expand.go:8` |
| M4 | Error messages leak internal paths/config | `server/api.go:33,49,75,82` |
| M5 | Resize values not validated (negative, zero, NaN) | `webtty/webtty.go:211-235` |
| M6 | Encoding change allows mid-session protocol switch | `webtty/webtty.go:203-209` |
| M7 | No connection timeout for WebSocket handshake | `server/server.go:75-78` |

### LOW (5)

| # | Finding | File |
|---|---------|------|
| L1 | Deprecated `gorilla/websocket` (archived 2023) | `go.mod:8` |
| L2 | `Server` header reveals software identity | `server/middleware.go:21` |
| L3 | Credential passed as CLI argument (visible in ps) | `main.go:38` |
| L4 | `titleVariables` can panic on bad config | `server/handlers.go:184` |
| L5 | No input size limit in `handleMasterReadEvent` | `webtty/webtty.go:95-97` |

---

## 4. Go Backend Test Gaps

### P0 — Critical (100% coverage required)

| File | Existing Tests | Missing Tests |
|------|---------------|---------------|
| `server/handlers.go` | None | WS auth, connection lifecycle, once-mode, max connections |
| `server/server.go` | None | Server creation, TLS config, origin checking, graceful shutdown |
| `server/ws_wrapper.go` | None | WS I/O adaptation, buffer overflow detection |
| `webtty/webtty.go` | Init, write, ping, resize | SetEncoding, unknown msg, malformed resize, permitWrite=false, context cancellation |
| `webtty/codecs.go` | None | Codec switching (base64/null) |

### P1 — High (90% coverage required)

| File | Existing Tests | Missing Tests |
|------|---------------|---------------|
| `server/api.go` | None | All 5 REST handlers |
| `server/middleware.go` | None | Basic auth, header injection, logging wrapper |
| `server/handler_atomic.go` | None | Connection counter, timer reset, concurrency |
| `server/options.go` | None | Validate() edge cases |
| `backend/tmux/sessions.go` | None | 14 exported session management functions |
| `backend/tmux/tmux.go` | None | TmuxSlave lifecycle |
| `backend/tmux/factory.go` | None | Param extraction, defaults |
| `backend/zellij/` (entire) | None | All functions |
| `backend/localcommand/local_command.go` | Factory, read/write | Close+timeout, ResizeTerminal, error paths |

### P2 — Medium (50% coverage required)

| File | Notes |
|------|-------|
| `pkg/homedir/expand.go` | **Bug:** panics on strings < 2 chars |
| `pkg/randomstring/generate.go` | Output properties unverified |
| `server/run_option.go`, `list_address.go`, `log_response_writer.go` | Low complexity |

---

## 5. Frontend Test Gaps

### P0 — TerminalTab WebSocket Protocol (CRITICAL)

The `ws.onmessage` handler that decodes base64 and writes to xterm is **completely untested**. This is the #1 critical data path.

**Missing unit tests:**
- WS message handler for type '1' (output) — base64 decode + terminal write
- WS message handler for type '3' (SetWindowTitle)
- `term.onData` callback — keystroke base64 encode + send as type '1'
- `term.onResize` callback — resize as type '3' JSON
- `ws.onerror` handler — error status display
- Base64 decode failure fallback (`atob` catch)
- Non-string message filtering
- Cleanup on unmount (ResizeObserver, ws.close, term.dispose)
- Re-connection on prop change (session/pane)

### P1 — Sidebar Session Management

**Missing unit tests:**
- `handleKill` (DELETE session)
- Polling interval creation/cleanup
- Create via "+" button click (only Enter key tested)
- `fetchDetail` caching and error case
- Empty input guard
- Pane click (`onSelect`)
- Session re-collapse

---

## 6. E2E & Test Infrastructure

### TEA Quality Scores

| Dimension | Score | Key Issues |
|-----------|-------|------------|
| **Deterministic** | 4/10 | `sleep 1` used 6 times across shell scripts, WebSocket tests loop with timeout |
| **Isolated** | 3/10 | Hardcoded session names, hardcoded port 18899, no parallel execution, shared state in auth.spec.ts |
| **Explicit** | 7/10 | Good test names and assertions; auth spec bundles 5 ops in one test |
| **Focused** | 6/10 | `test_e2e.sh` is a 407-line monolith; shell scripts overlap with Playwright |
| **Fast** | 6/10 | Server starts/stops 3x in shell E2E; estimated 30-60s total |

### What's Missing

- No CI/CD pipeline (no GitHub Actions, GitLab CI, or Makefile)
- No Docker/containerized test environment
- 3 overlapping E2E systems (Playwright, shell E2E, CLI wrapper)
- Python `websocket-client` dependency undeclared
- No port randomization for parallel execution
- Last test run shows failure with empty failure list (indeterminate)

---

## 7. Recommended Actions

### Phase 1 — Security Hardening (immediate, unblocks release)

1. Input sanitization: whitelist session names in `server/api.go`, `server/handlers.go`, tmux/zellij backends
2. Defaults audit: `PermitWrite` → `false`, require explicit opt-in for API access without auth
3. Origin check: default `CheckOrigin` to restrictive function
4. Timing-safe comparison: `subtle.ConstantTimeCompare` in `handlers.go:108`
5. Fix `homedir.Expand()` panic bug (`pkg/homedir/expand.go:8`)

### Phase 2 — Critical Test Coverage

6. WebSocket protocol tests (`webtty/webtty.go`): SetEncoding, unknown msg, malformed resize
7. Server lifecycle tests (`server/server.go`, `handlers.go`): WS upgrade, auth, once-mode
8. API endpoint tests (`server/api.go`): all 5 REST handlers, table-driven
9. Frontend WS message handler tests (`TerminalTab.tsx`): actual data path coverage
10. `server/ws_wrapper.go`: buffer overflow, I/O adaptation

### Phase 3 — Infrastructure

11. Consolidate E2E: remove shell E2E scripts, keep Playwright only
12. Add CI: GitHub Actions matrix (Go test / Vitest / Playwright E2E)
13. Fix isolation: unique ports, unique session names, proper cleanup
14. Add Makefile: `test`, `test-e2e`, `test-all`, `lint`

### Phase 4 — Hardening & Expansion

15. Rate limiting on auth endpoints
16. Security headers middleware (CSP, X-Frame-Options, HSTS)
17. TLS hardening (MinVersion 1.2, cipher suites, timeouts)
18. Backend test coverage: tmux, zellij, localcommand edge cases
19. Migrate from archived `gorilla/websocket` to community fork

---

## 8. TEA Engagement Recommendation

**TEA Integrated (Brownfield)** — the codebase is existing, non-trivial, and security-critical.

**Workflow:**
```
trace (baseline) → test-design (system-level risk) →
  per-feature: atdd → automate → test-review →
    trace (coverage + gate) → nfr-assess
```

**Gate Decision:** **CONCERNS** — proceed with documented mitigations for all score-6+ risks.
