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
    const header = page.locator('h3:has-text("Sessions")');
    await expect(header).toBeVisible();
  });

  test('sidebar has new session input', async ({ page }) => {
    await page.goto('/');
    const input = page.locator('input[placeholder="new session"]');
    await expect(input).toBeVisible();
  });

  test('xterm terminal container renders', async ({ page }) => {
    await page.goto('/');
    const xterm = page.locator('.xterm').first();
    await expect(xterm).toBeVisible({ timeout: 10000 });
  });

  test('status bar shows connected state', async ({ page }) => {
    await page.goto('/');
    const statusBar = page.locator('text=connected').first();
    await expect(statusBar).toBeVisible({ timeout: 10000 });
  });

  test('default tab ttyweb is shown in tab bar', async ({ page }) => {
    await page.goto('/');
    const tab = page.locator('text=ttyweb').first();
    await expect(tab).toBeVisible({ timeout: 5000 });
  });

  test('toggle sidebar hides and shows sidebar', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 5000 });

    const toggleBtn = page.locator('button').filter({ hasText: /[\u25C0\u25B6]/ }).first();
    await expect(toggleBtn).toBeVisible();

    // Click to hide sidebar
    await toggleBtn.click({ force: true });
    const header = page.locator('h3:has-text("Sessions")');
    await expect(header).not.toBeVisible();

    // Click to show sidebar again
    const toggleBtnAfter = page.locator('button').filter({ hasText: /[\u25C0\u25B6]/ }).first();
    await expect(toggleBtnAfter).toBeVisible();
    await toggleBtnAfter.click({ force: true });
    await expect(header).toBeVisible();
  });

  test('tab close button removes the tab', async ({ page }) => {
    await page.goto('/');

    // Wait for the default "ttyweb" tab to appear
    const tabLabel = page.locator('text=ttyweb').first();
    await expect(tabLabel).toBeVisible({ timeout: 5000 });

    // Find the close button next to it using XPath
    const closeBtn = tabLabel.locator('xpath=following-sibling::button').first();
    await closeBtn.click({ force: true });

    // The tab should be removed
    await expect(page.locator('text=ttyweb')).not.toBeVisible({ timeout: 3000 });
  });

  test('window resize does not crash page', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 5000 });

    // Resize window multiple times — no hard waits needed between resizes
    await page.setViewportSize({ width: 800, height: 600 });
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.setViewportSize({ width: 1024, height: 768 });

    // Page should still be functional
    const root = page.locator('#root');
    await expect(root).toBeAttached();
    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible();
  });
});
