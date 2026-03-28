import { test, expect } from './fixtures';
import {
  hasSessionManagement,
  cleanupSessions,
  waitForSessionInSidebar,
  waitForSessionRemovedFromSidebar,
} from './helpers';

test.describe('Session management via UI', () => {
  const createdSessions: string[] = [];

  test.afterEach(async ({ apiRequest }) => {
    await cleanupSessions(apiRequest, createdSessions);
    createdSessions.length = 0;
  });

  test('sidebar always shows Sessions header', async ({ page }) => {
    await page.goto('/');
    const header = page.locator('h3:has-text("Sessions")');
    await expect(header).toBeVisible({ timeout: 5000 });
  });

  test.describe.configure({ mode: 'serial' });

  test('sidebar displays session list from API', async ({ page, apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

    const name = `ui-list-${Date.now()}`;
    createdSessions.push(name);

    await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });

    await page.goto('/');
    await waitForSessionInSidebar(page, name);
  });

  test('create session via sidebar input', async ({ page, apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

    await page.goto('/');

    const name = `ui-create-${Date.now()}`;
    createdSessions.push(name);

    const input = page.locator('input[placeholder="new session"]');
    await expect(input).toBeVisible();
    await input.fill(name);
    await input.press('Enter');

    await waitForSessionInSidebar(page, name);
  });

  test('kill session via sidebar button', async ({ page, apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

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

    await waitForSessionRemovedFromSidebar(page, name);

    // Session successfully killed — remove from cleanup list
    const idx = createdSessions.indexOf(name);
    if (idx !== -1) createdSessions.splice(idx, 1);
  });

  test('expand session to show pane details', async ({ page, apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

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
    const sessionRow = sessionEntry.locator('..');
    await sessionRow.click({ force: true });

    const expandedIcon = page.locator('text=\u25BC').first();
    await expect(expandedIcon).toBeVisible({ timeout: 5000 });

    const connectLink = page.locator('text=Connect to session').first();
    await expect(connectLink).toBeVisible({ timeout: 5000 });
  });
});
