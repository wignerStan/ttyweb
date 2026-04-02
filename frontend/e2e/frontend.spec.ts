import { test, expect } from './fixtures';

test.describe('SPA frontend rendering', () => {
  test('page loads and renders #root with content', async ({ page }) => {
    await page.goto('/');
    const root = page.locator('#root');
    await expect(root).toBeAttached();
    expect(await root.evaluate((el) => el.children.length)).toBeGreaterThan(0);
  });

  test('page title is ttyweb', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle('ttyweb');
  });

  test('sidebar renders with Sessions header', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Sessions' })).toBeVisible();
  });

  test('sidebar has new session input', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByPlaceholder('new session')).toBeVisible();
  });

  test('xterm terminal container renders', async ({ page }) => {
    await page.goto('/');
    const xterm = page.locator('.xterm').first();
    await expect(xterm).toBeVisible({ timeout: 10000 });
  });

  test('status bar shows connected state', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('connected').first()).toBeVisible({ timeout: 10000 });
  });

  test('default tab ttyweb is shown in tab bar', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('ttyweb').first()).toBeVisible({ timeout: 5000 });
  });

  test('toggle sidebar hides and shows sidebar', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'Sessions' })).toBeVisible({ timeout: 5000 });

    // Deterministic locator — single strategy, no count() race
    const sidebarToggle = page.locator('button').filter({ hasText: /[\u25C0\u25B6]/ }).first();
    await expect(sidebarToggle).toBeVisible();

    // Click to hide sidebar
    await sidebarToggle.click();
    const header = page.getByRole('heading', { name: 'Sessions' });
    await expect(header).not.toBeVisible();

    // Click to show sidebar again
    await sidebarToggle.click();
    await expect(header).toBeVisible();
  });

  test('tab close button removes the tab', async ({ page }) => {
    await page.goto('/');

    // Wait for the default "ttyweb" tab to appear
    const tabLabel = page.getByText('ttyweb').first();
    await expect(tabLabel).toBeVisible({ timeout: 5000 });

    // Navigate to parent tab button, then find close button (sibling of span)
    const closeBtn = tabLabel.locator('..').getByRole('button', { name: 'Close tab' });
    await closeBtn.click();

    // The tab should be removed
    await expect(page.getByText('ttyweb')).not.toBeVisible({ timeout: 3000 });
  });

  test('window resize does not crash page', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'Sessions' })).toBeVisible({ timeout: 5000 });

    // Resize window multiple times
    await page.setViewportSize({ width: 800, height: 600 });
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.setViewportSize({ width: 1024, height: 768 });

    // Page should still be functional
    const root = page.locator('#root');
    await expect(root).toBeAttached();
    await expect(page.getByRole('heading', { name: 'Sessions' })).toBeVisible();
  });
});
