import { test, expect } from './fixtures';
import { hasSessionManagement } from './helpers';

test.describe('REST API', () => {
  test('GET /api/sessions returns success', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/sessions',
    });
    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('GET /api/backends returns backend list', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/backends',
    });
    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.length).toBeGreaterThan(0);
    const local = body.data.find((b: { name: string }) => b.name === 'local');
    expect(local).toBeDefined();
    expect(local.available).toBe(true);
  });

  test('POST /api/sessions creates a session', async ({ apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

    const sessionName = `pw-test-${Date.now()}`;

    const { status, body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name: sessionName },
    });
    expect(status).toBe(200);
    expect(createBody.success).toBe(true);
    expect(createBody.data.name).toBe(sessionName);

    // Verify it appears in list
    const { body: listBody } = await apiRequest({
      method: 'GET',
      path: '/api/sessions',
    });
    const names = listBody.data.map((s: { name: string }) => s.name);
    expect(names).toContain(sessionName);

    // Cleanup
    const { body: deleteBody } = await apiRequest({
      method: 'DELETE',
      path: `/api/sessions/${encodeURIComponent(sessionName)}`,
    });
    expect(deleteBody.success).toBe(true);
  });

  test('GET /api/sessions/{name} returns session detail', async ({ apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

    const sessionName = `pw-detail-${Date.now()}`;

    // Create session first
    await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name: sessionName },
    });

    // Get detail
    const { status, body } = await apiRequest({
      method: 'GET',
      path: `/api/sessions/${encodeURIComponent(sessionName)}`,
    });
    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.name).toBe(sessionName);

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/sessions/${encodeURIComponent(sessionName)}`,
    });
  });

  test('GET /api/sessions/nonexistent returns 404', async ({ apiRequest }) => {
    // Local backend returns 503 (session management unavailable), tmux/zellij returns 404
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/sessions/nonexistent-xyz-999',
      retryConfig: { maxRetries: 0 },
    });
    expect([404, 503]).toContain(status);
  });

  test('POST with invalid JSON returns 400', async ({ request }) => {
    // apiRequest serializes body as JSON — use raw Playwright request for invalid JSON
    const res = await request.fetch('/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json',
    });
    // Local backend returns 503 before checking JSON (session management unavailable)
    // Session-managing backends return 400 (invalid JSON)
    expect([400, 503]).toContain(res.status());
  });

  test('PUT /api/sessions returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'PUT',
      path: '/api/sessions',
    });
    expect(status).toBe(405);
  });
});
