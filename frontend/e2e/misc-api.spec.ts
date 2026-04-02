import { test, expect } from './fixtures';

// These API endpoint tests were previously in mobile.spec.ts but are pure
// API tests unrelated to mobile UI. They verify miscellaneous endpoints.

test.describe('Misc API endpoints', () => {
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
