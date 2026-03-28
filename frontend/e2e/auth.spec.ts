import { test, expect } from './fixtures';
import { hasSessionManagement } from './helpers';

test.describe.configure({ mode: 'serial' });

test.describe('Basic auth', () => {
  test('API returns expected response structure', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/sessions',
    });
    expect(status).toBe(200);
    expect(body).toHaveProperty('success');
    expect(body).toHaveProperty('data');
    expect(typeof body.success).toBe('boolean');
  });

  test('session CRUD lifecycle via API', async ({ apiRequest, request }) => {
    test.skip(!await hasSessionManagement(request), 'Requires session management');

    const name = `auth-crud-${Date.now()}`;

    // Create
    const { status: createStatus, body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });
    expect(createStatus).toBe(200);
    expect(createBody.success).toBe(true);

    // Read
    const { status: readStatus, body: readBody } = await apiRequest({
      method: 'GET',
      path: `/api/sessions/${encodeURIComponent(name)}`,
    });
    expect(readStatus).toBe(200);
    expect(readBody.success).toBe(true);
    expect(readBody.data.name).toBe(name);

    // Verify in list
    const { body: listBody } = await apiRequest({
      method: 'GET',
      path: '/api/sessions',
    });
    expect(listBody.data.some((s: { name: string }) => s.name === name)).toBe(true);

    // Delete
    const { status: delStatus, body: delBody } = await apiRequest({
      method: 'DELETE',
      path: `/api/sessions/${encodeURIComponent(name)}`,
    });
    expect(delStatus).toBe(200);
    expect(delBody.success).toBe(true);

    // Verify gone from list
    const { body: listAfterBody } = await apiRequest({
      method: 'GET',
      path: '/api/sessions',
    });
    expect(listAfterBody.data.some((s: { name: string }) => s.name === name)).toBe(false);
  });

  test('invalid session name returns appropriate error', async ({ apiRequest }) => {
    // Local backend returns 503 (session management unavailable), tmux/zellij returns 404
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/sessions/nonexistent-session-xyz',
      retryConfig: { maxRetries: 0 },
    });
    expect([404, 503]).toContain(status);
  });

  test('wrong HTTP method returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'PATCH',
      path: '/api/sessions/test',
      retryConfig: { maxRetries: 0 },
    });
    // Local backend returns 503 (no session manager), tmux/zellij returns 405
    expect([405, 503]).toContain(status);
  });
});
