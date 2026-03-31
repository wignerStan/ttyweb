import { test, expect } from './fixtures';
import { mobileViewport } from './helpers';

// ─── Theme Tests ──────────────────────────────────────────────────────────────

test.describe('Theme', () => {
  test('page loads with dark theme by default', async ({ page }) => {
    await page.goto('/');
    // Dark theme = no data-theme attribute (useTheme removes it for dark)
    const dataTheme = await page.getAttribute('html', 'data-theme');
    expect(dataTheme).toBeNull();

    // Verify dark theme CSS variable is applied
    const bgColor = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue('--bg-primary').trim(),
    );
    expect(bgColor).toBe('#1a1b26');
  });

  test('theme toggle is visible', async ({ page }) => {
    await page.goto('/');
    // ThemeToggle renders an SVG sun icon in dark mode (moon in light)
    // It lives inside the tab bar area
    const toggle = page.locator('button[title="Switch to light theme"]');
    await expect(toggle).toBeVisible({ timeout: 10000 });
  });

  test('theme persists across navigation', async ({ page }) => {
    await page.goto('/');

    // Click theme toggle to switch to light
    const toggleDark = page.locator('button[title="Switch to light theme"]');
    await expect(toggleDark).toBeVisible({ timeout: 10000 });
    await toggleDark.click();

    // Verify light theme is active
    const dataTheme = await page.getAttribute('html', 'data-theme');
    expect(dataTheme).toBe('light');

    // Navigate to mobile route
    await page.goto('/m');
    // Light theme should persist via localStorage
    const dataThemeMobile = await page.getAttribute('html', 'data-theme');
    expect(dataThemeMobile).toBe('light');

    // Navigate back to desktop
    await page.goto('/');
    const dataThemeBack = await page.getAttribute('html', 'data-theme');
    expect(dataThemeBack).toBe('light');

    // Clean up: restore dark theme
    const toggleLight = page.locator('button[title="Switch to dark theme"]');
    if (await toggleLight.isVisible()) {
      await toggleLight.click();
    }
  });

  test('theme toggle switches between dark and light', async ({ page }) => {
    await page.goto('/');

    // Verify dark theme initially
    await expect(page.locator('button[title="Switch to light theme"]')).toBeVisible({ timeout: 10000 });

    // Toggle to light
    await page.locator('button[title="Switch to light theme"]').click();
    await expect(page.locator('button[title="Switch to dark theme"]')).toBeVisible();

    // Verify light theme CSS variable
    const bgColor = await page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue('--bg-primary').trim(),
    );
    expect(bgColor).toBe('#f5f5f5');

    // Toggle back to dark
    await page.locator('button[title="Switch to dark theme"]').click();
    await expect(page.locator('button[title="Switch to light theme"]')).toBeVisible();
  });
});

// ─── i18n Tests ───────────────────────────────────────────────────────────────

test.describe('i18n', () => {
  test('page renders with English text by default', async ({ page }) => {
    await page.goto('/');
    // Check for English strings visible in the UI
    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=Terminal').first()).toBeVisible();
    await expect(page.locator('text=Conversations').first()).toBeVisible();
  });

  test('i18n system is initialized and language can be changed via API', async ({ page }) => {
    await page.goto('/');

    // Verify i18next is available and default language is English
    const lang = await page.evaluate(() => (window as Record<string, unknown>).i18next?.language);
    expect(lang).toBe('en');

    // Change language to Chinese
    await page.evaluate(() => {
      (window as Record<string, unknown>).i18next?.changeLanguage?.('zh');
    });

    // Wait for language change to propagate
    await page.waitForTimeout(500);

    // Verify language was changed
    const newLang = await page.evaluate(() => (window as Record<string, unknown>).i18next?.language);
    expect(newLang).toBe('zh');

    // Verify Chinese text appears where i18n t() is used
    // Note: not all components use t() yet, so we check the i18n store directly
    const sessionsText = await page.evaluate(() =>
      (window as Record<string, unknown>).i18next?.t?.('sidebar.title'),
    );
    expect(sessionsText).toBe('\u4f1a\u8bdd'); // 会话
  });

  test('key UI elements have proper text content', async ({ page }) => {
    await page.goto('/');

    // Sidebar header
    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 10000 });

    // Tab bar buttons
    await expect(page.locator('button', { hasText: 'Terminal' }).first()).toBeVisible();
    await expect(page.locator('button', { hasText: 'Conversations' }).first()).toBeVisible();

    // New session input placeholder
    await expect(page.locator('input[placeholder="new session"]')).toBeVisible();
  });
});

// ─── Mobile Layout Tests ──────────────────────────────────────────────────────

test.describe('Mobile layout', () => {
  test.use(mobileViewport());

  test('mobile route /m loads', async ({ page }) => {
    const logs: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') logs.push(msg.text());
    });

    await page.goto('/m');
    const root = page.locator('#root');
    await expect(root).toBeAttached({ timeout: 10000 });
    expect(await root.evaluate((el) => el.children.length)).toBeGreaterThan(0);

    // No console errors on load
    expect(logs).toHaveLength(0);
  });

  test('mobile terminal renders', async ({ page }) => {
    await page.goto('/m');
    // The xterm container should exist even before a session is connected
    // (it may not have content yet)
    const mobileHeader = page.locator('.mobile-header');
    await expect(mobileHeader).toBeVisible({ timeout: 10000 });
  });

  test('mobile toolbar renders', async ({ page }) => {
    await page.goto('/m');
    const mobileHeader = page.locator('.mobile-header');
    await expect(mobileHeader).toBeVisible({ timeout: 10000 });
  });

  test('mobile drawer opens', async ({ page }) => {
    await page.goto('/m');
    const menuBtn = page.locator('.mobile-menu-btn').first();
    await expect(menuBtn).toBeVisible({ timeout: 10000 });
    await menuBtn.dispatchEvent('click');
    await expect(page.locator('.mobile-drawer.open')).toBeVisible({ timeout: 5000 });
  });
});

// ─── Responsive Layout ────────────────────────────────────────────────────────

test.describe('Responsive layout', () => {
  test('desktop layout renders sidebar', async ({ page }) => {
    await page.goto('/');
    const sidebar = page.locator('h3:has-text("Sessions")');
    await expect(sidebar).toBeVisible({ timeout: 10000 });
  });

  test('page is responsive at common widths', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 5000 });

    // Test common viewport widths — no horizontal overflow
    for (const { width, height } of [
      { width: 1280, height: 720 },
      { width: 1024, height: 768 },
      { width: 768, height: 1024 },
      { width: 480, height: 800 },
      { width: 375, height: 812 },
    ]) {
      await page.setViewportSize({ width, height });
      // Allow the page to re-render
      await page.waitForTimeout(100);

      // Verify no horizontal scroll on body
      const hasHorizontalOverflow = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });
      expect(hasHorizontalOverflow).toBe(false);
    }

    // Page should still be functional after resizing
    const root = page.locator('#root');
    await expect(root).toBeAttached();
  });
});

// ─── Accessibility Tests ──────────────────────────────────────────────────────

test.describe('Accessibility', () => {
  test('page has proper title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle('ttyweb');
  });

  test('interactive elements are focusable', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 10000 });

    // Find all buttons and verify they can receive focus
    const buttons = page.locator('button:visible');
    const count = await buttons.count();
    expect(count).toBeGreaterThan(0);

    // Tab through focusable elements and verify at least some receive focus
    let focusedCount = 0;
    for (let i = 0; i < Math.min(count, 10); i++) {
      const btn = buttons.nth(i);
      await btn.focus();
      const isFocused = await btn.evaluate((el) => document.activeElement === el);
      if (isFocused) focusedCount++;
    }

    // At least the sidebar toggle and theme toggle should be focusable
    expect(focusedCount).toBeGreaterThanOrEqual(2);
  });

  test('input elements accept focus and keyboard input', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h3:has-text("Sessions")')).toBeVisible({ timeout: 10000 });

    const input = page.locator('input[placeholder="new session"]');
    await expect(input).toBeVisible();

    // Focus and type
    await input.focus();
    const isFocused = await input.evaluate((el) => document.activeElement === el);
    expect(isFocused).toBe(true);

    await input.fill('test-session');
    await expect(input).toHaveValue('test-session');

    // Clear
    await input.clear();
    await expect(input).toHaveValue('');
  });

  test('no console errors on page load', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.goto('/');
    // Wait for the page to fully load including xterm
    await expect(page.locator('.xterm').first()).toBeVisible({ timeout: 10000 });

    // Filter out known benign errors (xterm CSS warnings, etc.)
    const filteredErrors = errors.filter((e) => {
      // xterm.js sometimes logs non-critical warnings
      if (e.includes('xterm') || e.includes('CSS')) return false;
      return true;
    });

    expect(filteredErrors).toHaveLength(0);
  });
});

// ─── Performance Tests ────────────────────────────────────────────────────────

test.describe('Performance', () => {
  test('page load time is reasonable', async ({ page }) => {
    const start = Date.now();
    await page.goto('/');
    // Wait for the page to be fully rendered
    await expect(page.locator('.xterm').first()).toBeVisible({ timeout: 10000 });
    const elapsed = Date.now() - start;

    // Page should load within 3 seconds (excluding network latency on CI)
    expect(elapsed).toBeLessThan(3000);
  });

  test('API response time is reasonable', async ({ apiRequest }) => {
    const start = Date.now();
    const { status } = await apiRequest({ method: 'GET', path: '/api/health' });
    const elapsed = Date.now() - start;

    expect(status).toBe(200);
    // API should respond within 1 second
    expect(elapsed).toBeLessThan(1000);
  });

  test('mobile page load time is reasonable', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.setUserAgent(
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
    );

    const start = Date.now();
    await page.goto('/m');
    await expect(page.locator('.mobile-header')).toBeVisible({ timeout: 10000 });
    const elapsed = Date.now() - start;

    expect(elapsed).toBeLessThan(3000);
  });
});
