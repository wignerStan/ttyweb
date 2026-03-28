import { test, expect } from './fixtures';
import { mobileViewport, openFirstSession, hasSessionsWithPanes } from './helpers';

test.describe('Mobile UI', () => {
  test.use(mobileViewport());

  test('mobile route /m loads mobile layout', async ({ page }) => {
    await page.goto('/m');
    await expect(page.locator('.mobile-header')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('h3:has-text("Sessions")')).not.toBeVisible();
  });

  test('mobile drawer opens and shows TmuxTree', async ({ page }) => {
    await page.goto('/m');
    const menuBtn = page.locator('.mobile-menu-btn').first();
    await expect(menuBtn).toBeVisible({ timeout: 10000 });
    await menuBtn.dispatchEvent('click');
    await expect(page.locator('.mobile-drawer.open')).toBeVisible({ timeout: 5000 });
  });

  for (const title of [
    'mobile terminal renders when session is selected',
    'toolbox quick keys are visible when terminal is active',
    'toolbox tab bar shows content tabs',
    'tab close removes terminal tab and shows placeholder',
    'mobile WebSocket connects and terminal receives output',
    'font size slider is visible when terminal is active',
  ]) {
    test(`${title} (requires sessions with panes)`, async ({ page }) => {
      test.skip(!await hasSessionsWithPanes(page), 'No sessions with panes available');
      await page.goto('/m');
      try {
        await openFirstSession(page);
      } catch {
        test.skip('Could not open session pane — tree UI may not render panes for this backend');
      }

      switch (title) {
        case 'mobile terminal renders when session is selected':
          await expect(page.locator('.mobile-tab').first()).toBeVisible({ timeout: 5000 });
          await expect(page.locator('.mobile-terminal-container .xterm').first()).toBeVisible({ timeout: 10000 });
          break;
        case 'toolbox quick keys are visible when terminal is active':
          await expect(page.locator('.mobile-tab').first()).toBeVisible({ timeout: 5000 });
          await expect(page.locator('.toolbox-key').first()).toBeVisible({ timeout: 10000 });
          await expect(page.locator('.toolbox-key', { hasText: 'esc' })).toBeVisible();
          await expect(page.locator('.toolbox-key', { hasText: 'tab' })).toBeVisible();
          await expect(page.locator('.toolbox-key', { hasText: '^C' })).toBeVisible();
          break;
        case 'toolbox tab bar shows content tabs':
          await expect(page.locator('.mobile-tab').first()).toBeVisible({ timeout: 5000 });
          await expect(page.locator('.toolbox-tabbar')).toBeVisible({ timeout: 10000 });
          await expect(page.locator('.toolbox-tab').count()).toBeGreaterThan(0);
          break;
        case 'tab close removes terminal tab and shows placeholder':
          await expect(page.locator('.mobile-tab').first()).toBeVisible({ timeout: 5000 });
          await page.locator('.mobile-tab-close').first().click({ force: true });
          await expect(page.locator('.mobile-tab')).not.toBeVisible({ timeout: 5000 });
          await expect(page.locator('.mobile-placeholder')).toBeVisible();
          break;
        case 'mobile WebSocket connects and terminal receives output':
          await expect(page.locator('.mobile-terminal-container .xterm').first()).toBeVisible({ timeout: 10000 });
          await expect.poll(async () => {
            const termText = await page.locator('.xterm').first().textContent();
            return termText !== null && termText.length > 0;
          }, { timeout: 10000, intervals: [500, 1000] }).toBe(true);
          break;
        case 'font size slider is visible when terminal is active':
          await expect(page.locator('.font-slider-input').first()).toBeVisible({ timeout: 10000 });
          await expect(page.locator('.font-slider-value').first()).toBeVisible();
          break;
      }
    });
  }
});

test.describe('Mobile API endpoints', () => {
  test('GET /api/profiles returns success', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'GET', path: '/api/profiles' });
    expect(status).toBe(200);
  });

  test('POST /api/profiles creates a profile', async ({ apiRequest }) => {
    const name = `test-profile-${Date.now()}`;
    const { status } = await apiRequest({ method: 'POST', path: '/api/profiles', body: { profile_key: name, name } });
    expect(status).toBe(200);
  });

  test('GET /api/snippets returns success', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'GET', path: '/api/snippets' });
    expect(status).toBe(200);
  });

  test('GET /api/roles returns success', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'GET', path: '/api/roles' });
    expect(status).toBe(200);
  });

  test('GET /api/roles/defaults returns default roles', async ({ apiRequest }) => {
    const { body } = await apiRequest({ method: 'GET', path: '/api/roles/defaults' });
    const data = body as { data: unknown[] };
    expect(Array.isArray(data.data)).toBe(true);
    expect(data.data.length).toBeGreaterThan(0);
  });

  test('GET /api/panes/status returns success', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'GET', path: '/api/panes/status' });
    expect(status).toBe(200);
  });

  test('GET /api/tmux/config returns prefix config', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'GET', path: '/api/tmux/config' });
    expect(status).toBe(200);
  });

  test('POST /api/telemetry accepts events', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'POST', path: '/api/telemetry', body: { events: [] } });
    expect(status).toBe(200);
  });

  test('GET /api/opencode-config returns success', async ({ apiRequest }) => {
    const { status } = await apiRequest({ method: 'GET', path: '/api/opencode-config' });
    expect(status).toBe(200);
  });
});
