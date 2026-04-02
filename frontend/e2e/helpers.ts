import { expect, type Page, type APIRequestContext } from '@playwright/test';

// ─── Backend detection ─────────────────────────────────────────────────────

interface BackendInfo {
  name: string;
  available: boolean;
  active: boolean;
}

async function fetchActiveBackend(request: APIRequestContext): Promise<BackendInfo | null> {
  try {
    const res = await request.fetch('/api/backends');
    const data = await res.json();
    const backends = data.data as BackendInfo[];
    return backends.find((b) => b.active) ?? null;
  } catch {
    return null;
  }
}

/** Check whether the active backend supports session management (tmux or zellij). */
export async function hasSessionManagement(request: APIRequestContext): Promise<boolean> {
  const active = await fetchActiveBackend(request);
  if (!active || active.name === 'local') return false;
  return active.available;
}

// ─── Session CRUD helpers ──────────────────────────────────────────────────

type ApiRequestFn = (opts: {
  method: string;
  path: string;
  body?: Record<string, unknown>;
  retryConfig?: { maxRetries: number };
}) => Promise<{ status: number; body: Record<string, unknown> }>;

/** Delete a session via the API. Ignores errors (session may already be gone). */
export async function deleteSession(
  apiRequest: ApiRequestFn,
  name: string,
): Promise<void> {
  try {
    await apiRequest({
      method: 'DELETE',
      path: `/api/sessions/${encodeURIComponent(name)}`,
      retryConfig: { maxRetries: 0 },
    });
  } catch {
    // Session may already be deleted
  }
}

/** Batch-delete sessions. Intended for afterEach/afterAll hooks. */
export async function cleanupSessions(
  apiRequest: ApiRequestFn,
  names: readonly string[],
): Promise<void> {
  await Promise.all(names.map((name) => deleteSession(apiRequest, name)));
}

// ─── Sidebar DOM polling ───────────────────────────────────────────────────

/**
 * Wait for a session name to appear in the sidebar.
 * Targets data-testid="session-name" (Sidebar) and .session-name (TmuxTree).
 */
export async function waitForSessionInSidebar(page: Page, name: string): Promise<void> {
  const locator = page.locator('[data-testid="session-name"], .session-name', { hasText: name });
  await expect(locator).toBeVisible({ timeout: 10000 });
}

/**
 * Wait for a session name to disappear from the sidebar.
 * Targets data-testid="session-name" (Sidebar) and .session-name (TmuxTree).
 */
export async function waitForSessionRemovedFromSidebar(page: Page, name: string): Promise<void> {
  const locator = page.locator('[data-testid="session-name"], .session-name', { hasText: name });
  await expect(locator).not.toBeVisible({ timeout: 10000 });
}

// ─── WebSocket monitoring ─────────────────────────────────────────────────

/**
 * Inject a WebSocket monitor that captures all sent/received messages
 * and stores the WS instance on `window.__wsInstance`.
 */
export async function injectWebSocketMonitor(page: Page): Promise<void> {
  await page.addInitScript(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const W = window as any;
    const origWebSocket = W.WebSocket;
    W.__wsMessages = [];
    W.__wsSent = [];
    W.WebSocket = function (url: string | URL, protocols?: string | string[]) {
      const ws = protocols ? new origWebSocket(url, protocols) : new origWebSocket(url);
      ws.addEventListener('message', (event: MessageEvent) => {
        W.__wsMessages.push(event.data);
      });
      const origSend = ws.send.bind(ws);
      ws.send = (data: string | ArrayBufferLike | Blob | ArrayBufferView) => {
        W.__wsSent.push(data);
        return origSend(data);
      };
      W.__wsInstance = ws;
      return ws;
    } as unknown as typeof WebSocket;
  });
}

// ─── Mobile helpers ───────────────────────────────────────────────────────

const MOBILE_VIEWPORT = { width: 375, height: 812 } as const;
const MOBILE_USER_AGENT =
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1';

/** Returns viewport and userAgent config for mobile tests. */
export function mobileViewport() {
  return {
    viewport: { ...MOBILE_VIEWPORT },
    userAgent: MOBILE_USER_AGENT,
  };
}

/**
 * Open the mobile drawer, click the first session row to expand panes,
 * then click the first pane to connect to a terminal.
 */
export async function openFirstSession(page: Page): Promise<void> {
  const menuBtn = page.locator('.mobile-menu-btn').first();
  await expect(menuBtn).toBeVisible({ timeout: 10000 });
  await menuBtn.click();

  await expect
    .poll(
      async () => page.locator('.session-row').count(),
      { timeout: 10000, intervals: [500, 1000] },
    )
    .toBeGreaterThan(0);

  await page.locator('.session-row').first().click();

  await expect
    .poll(
      async () => page.locator('.pane-node').count(),
      { timeout: 10000, intervals: [500, 1000] },
    )
    .toBeGreaterThan(0);

  await page.locator('.pane-node').first().click();
}

// ─── Tree API helpers ─────────────────────────────────────────────────────

interface TreeSession {
  windows?: Array<{ panes?: unknown[] }>;
}

/**
 * Fetch sessions from the tree API.
 * `/api/tmux/tree` uses the sessionManager interface (backend-agnostic despite the URL).
 */
async function fetchTreeSessions(page: Page): Promise<TreeSession[] | null> {
  try {
    const res = await page.request.get('/api/tmux/tree');
    if (!res.ok()) return null;
    const data = await res.json();
    const sessions = data.data?.sessions ?? data.data ?? data;
    return Array.isArray(sessions) ? sessions : null;
  } catch {
    return null;
  }
}

/** Check whether any sessions exist in the tree API. */
export async function hasSessionsInTree(page: Page): Promise<boolean> {
  const sessions = await fetchTreeSessions(page);
  return sessions !== null && sessions.length > 0;
}

/**
 * Check whether sessions exist with at least one pane.
 * For zellij, panes are nil so this returns false — mobile pane-clicking tests
 * should use this to skip gracefully.
 */
export async function hasSessionsWithPanes(page: Page): Promise<boolean> {
  const sessions = await fetchTreeSessions(page);
  return sessions !== null && sessions.some(
    (s) => s.windows?.some((w) => w.panes && w.panes.length > 0),
  );
}
