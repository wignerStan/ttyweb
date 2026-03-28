import { test, expect } from './fixtures';
import { injectWebSocketMonitor } from './helpers';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type W = any;

test.describe('WebSocket terminal connection', () => {
  test('WebSocket connects and terminal shows connected status', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('text=connected').first()).toBeVisible({ timeout: 10000 });
  });

  test('WebSocket receives terminal output', async ({ page }) => {
    await injectWebSocketMonitor(page);
    await page.goto('/');

    // Wait for WebSocket connection first to avoid race condition
    await expect(page.locator('text=connected').first()).toBeVisible({ timeout: 15000 });

    // Wait for terminal output (type '1' = base64 encoded output, type '3' = title)
    await page.waitForFunction(
      () => {
        const msgs = (window as W).__wsMessages || [];
        return msgs.some((msg: string) => msg[0] === '1' || msg[0] === '3');
      },
      { timeout: 15000 },
    );

    const wsMessages = await page.evaluate(() => (window as W).__wsMessages || []);

    expect(wsMessages.length).toBeGreaterThan(0);
    const hasOutputOrTitle = wsMessages.some((msg: string) => msg[0] === '1' || msg[0] === '3');
    expect(hasOutputOrTitle).toBe(true);
  });

  test('xterm terminal container renders after WS connection', async ({ page }) => {
    await page.goto('/');
    const xterm = page.locator('.xterm').first();
    await expect(xterm).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=connected').first()).toBeVisible({ timeout: 5000 });
  });

  test('WebSocket sends initial handshake and encoding messages', async ({ page }) => {
    await injectWebSocketMonitor(page);
    await page.goto('/');

    await expect(page.locator('text=connected').first()).toBeVisible({ timeout: 10000 });

    const sentMessages = await page.evaluate(() => (window as W).__wsSent || []);

    // Should have sent AuthToken init message and '4base64' encoding message
    expect(sentMessages.length).toBeGreaterThanOrEqual(2);
    expect(sentMessages.some((msg: string) => msg.includes('AuthToken'))).toBe(true);
    expect(sentMessages.some((msg: string) => msg === '4base64')).toBe(true);
  });

  test('status bar shows connection status', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('text=connected').first()).toBeVisible({ timeout: 10000 });

    const bodyText = await page.evaluate(() => document.body.textContent);
    expect(bodyText).toContain('connected');
  });

  test('WebSocket disconnect updates status to disconnected', async ({ page }) => {
    await injectWebSocketMonitor(page);
    await page.goto('/');

    // Wait for connected state
    await expect(page.locator('text=connected').first()).toBeVisible({ timeout: 10000 });

    // Programmatically close the WebSocket instance
    await page.evaluate(() => {
      const ws = (window as W).__wsInstance as WebSocket | null;
      if (ws) ws.close(1000, 'test disconnect');
    });

    // Verify status transitions to "disconnected"
    await expect(page.locator('text=disconnected').first()).toBeVisible({ timeout: 5000 });
  });
});
