import { test, expect } from './fixtures';
import { injectWebSocketMonitor } from './helpers';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type W = any;

test.describe('Terminal input and output', () => {
  test('WebSocket protocol correctly encodes input', async ({ page }) => {
    await injectWebSocketMonitor(page);
    await page.goto('/');

    // Wait for WebSocket connected and handshake sent
    await expect(page.getByText('connected').first()).toBeVisible({ timeout: 10000 });

    // Verify handshake messages were captured
    const sentMessages = await page.evaluate(() => (window as W).__wsSent || []);
    expect(sentMessages.length).toBeGreaterThanOrEqual(2);

    // Verify AuthToken handshake was sent
    const authTokenMsg = sentMessages.find((msg: string) => msg.includes('AuthToken'));
    expect(authTokenMsg).toBeDefined();

    // Verify base64 encoding preference was sent
    expect(sentMessages).toContain('4base64');
  });

  test('WebSocket correctly decodes output messages', async ({ page }) => {
    await injectWebSocketMonitor(page);
    await page.goto('/');

    // Wait for WebSocket connected
    await expect(page.getByText('connected').first()).toBeVisible({ timeout: 10000 });

    // Wait for output messages from the server
    await page.waitForFunction(
      () => {
        const msgs = (window as W).__wsMessages || [];
        return msgs.length > 0;
      },
      { timeout: 10000 },
    );

    const receivedMessages = await page.evaluate(() => (window as W).__wsMessages || []);
    expect(receivedMessages.length).toBeGreaterThan(0);

    // Verify message type prefixes — protocol defines types 0-6
    const validOutputPrefixes = ['0', '1', '2', '3', '4', '5', '6'];
    for (const msg of receivedMessages) {
      const type = String(msg)[0];
      expect(validOutputPrefixes).toContain(type);
    }
  });

  test('WebSocket message format follows protocol spec', async ({ page }) => {
    await injectWebSocketMonitor(page);
    await page.goto('/');

    await expect(page.getByText('connected').first()).toBeVisible({ timeout: 10000 });

    // Collect all sent and received messages
    const { sent, received } = await page.evaluate(() => ({
      sent: (window as W).__wsSent || [],
      received: (window as W).__wsMessages || [],
    }));

    // Sent messages: should include handshake and encoding preference
    expect(sent.length).toBeGreaterThanOrEqual(2);

    // All received messages should use valid protocol type prefixes (0-6)
    const validPrefixes = ['0', '1', '2', '3', '4', '5', '6'];
    for (const msg of received) {
      const prefix = String(msg)[0];
      expect(validPrefixes).toContain(prefix);
    }

    // Sent messages can be protocol-typed (0-4) or JSON (AuthToken handshake)
    const validSentPrefixes = ['0', '1', '2', '3', '4', '{'];
    for (const msg of sent) {
      const prefix = String(msg)[0];
      expect(validSentPrefixes).toContain(prefix);
    }

    // Verify full round-trip
    expect(sent.length).toBeGreaterThan(0);
    expect(received.length).toBeGreaterThan(0);
  });
});
