import { test, expect } from './fixtures';

test.describe('Session management via UI', () => {
  // Track sessions created during each test for guaranteed cleanup
  const createdSessions: string[] = [];

  test.afterEach(async ({ apiRequest }) => {
    for (const name of createdSessions) {
      try {
        await apiRequest({
          method: 'DELETE',
          path: `/api/sessions/${encodeURIComponent(name)}`,
        });
      } catch {
        // Session may already be deleted by the test itself — ignore
      }
    }
    createdSessions.length = 0;
  });

  // Wait for sidebar to display a session name by polling the DOM.
  // The sidebar fetches /api/sessions every 3s — we poll the DOM until the text appears.
  async function waitForSessionInSidebar(
    page: import('@playwright/test').Page,
    name: string,
  ) {
    await expect
      .poll(
        async () => {
          const elements = await page.locator('*').all();
          for (const el of elements) {
            const text = await el.textContent();
            if (text?.includes(name)) return true;
          }
          return false;
        },
        { timeout: 10000, intervals: [500, 1000] },
      )
      .toBe(true);
  }

  // Wait for a session to be removed from the sidebar by polling the DOM.
  async function waitForSessionRemovedFromSidebar(
    page: import('@playwright/test').Page,
    name: string,
  ) {
    await expect
      .poll(
        async () => {
          const elements = await page.locator('*').all();
          for (const el of elements) {
            const text = await el.textContent();
            if (text?.includes(name)) return false;
          }
          return true;
        },
        { timeout: 10000, intervals: [500, 1000] },
      )
      .toBe(true);
  }

  test('sidebar displays session list from API', async ({ page, apiRequest }) => {
    const name = `ui-list-${Date.now()}`;
    createdSessions.push(name);

    await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });

    await page.goto('/');
    await waitForSessionInSidebar(page, name);

    const sessionEntry = page.locator(`text=${name}`).first();
    await expect(sessionEntry).toBeVisible();
  });

  test('create session via sidebar input', async ({ page, apiRequest }) => {
    await page.goto('/');

    const name = `ui-create-${Date.now()}`;
    createdSessions.push(name);

    const input = page.locator('input[placeholder="new session"]');
    await expect(input).toBeVisible();
    await input.fill(name);
    await input.press('Enter');

    await waitForSessionInSidebar(page, name);

    const sessionEntry = page.locator(`text=${name}`).first();
    await expect(sessionEntry).toBeVisible();
  });

  test('kill session via sidebar button', async ({ page, apiRequest }) => {
    const name = `ui-kill-${Date.now()}`;
    createdSessions.push(name);

    await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });

    await page.goto('/');
    await waitForSessionInSidebar(page, name);

    const sessionEntry = page.locator(`text=${name}`).first();
    await expect(sessionEntry).toBeVisible();

    const sessionRow = sessionEntry.locator('..');
    const killBtn = sessionRow.locator('button[title="Kill session"]');
    await killBtn.click({ force: true });

    // Wait for sidebar to refresh and verify session is gone via API
    await expect
      .poll(async () => {
        const { body } = await apiRequest({ method: 'GET', path: '/api/sessions' });
        return body.data.some((s: { name: string }) => s.name === name);
      })
      .toBe(false);

    // Verify in UI as well
    await waitForSessionRemovedFromSidebar(page, name);

    // Session successfully killed — remove from cleanup list
    const idx = createdSessions.indexOf(name);
    if (idx !== -1) createdSessions.splice(idx, 1);
  });

  test('expand session to show pane details', async ({ page, apiRequest }) => {
    const name = `ui-expand-${Date.now()}`;
    createdSessions.push(name);

    await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });

    await page.goto('/');
    await waitForSessionInSidebar(page, name);

    const sessionEntry = page.locator(`text=${name}`).first();
    await expect(sessionEntry).toBeVisible();

    const sessionRow = sessionEntry.locator('..');
    await sessionRow.click({ force: true });

    const expandedIcon = page.locator('text=\u25BC').first();
    await expect(expandedIcon).toBeVisible({ timeout: 5000 });

    const connectLink = page.locator('text=Connect to session').first();
    await expect(connectLink).toBeVisible({ timeout: 5000 });
  });

  test('sidebar always shows Sessions header', async ({ page }) => {
    await page.goto('/');
    const header = page.locator('h3:has-text("Sessions")');
    await expect(header).toBeVisible({ timeout: 5000 });
  });
});
