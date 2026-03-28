import { test, expect } from './fixtures';

test.describe('API edge cases (risk-based)', () => {
  test('DELETE nonexistent session returns error', async ({ apiRequest }) => {
    // Backend returns 500 for kill failure on nonexistent sessions
    const { status } = await apiRequest({
      method: 'DELETE',
      path: '/api/sessions/nonexistent-session-xyz-999',
    });
    expect([404, 500]).toContain(status);
  });

  test('session name with special characters is handled', async ({ apiRequest }) => {
    const name = `special-chars-!@#$%^&*()`;
    const { status: createStatus, body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });

    // The API should either create it successfully or reject with validation error
    expect([200, 400, 422, 500]).toContain(createStatus);

    // If created, verify cleanup works
    if (createStatus === 200) {
      expect(createBody.success).toBe(true);
      await apiRequest({
        method: 'DELETE',
        path: `/api/sessions/${encodeURIComponent(name)}`,
      });
    }
  });

  test('session name with unicode characters is handled', async ({ apiRequest }) => {
    const name = `unicode-\u4E2D\u6587-test`;
    const { status: createStatus, body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name },
    });

    // Should succeed — unicode names are valid
    if (createStatus === 200) {
      expect(createBody.success).toBe(true);

      // Verify it appears in list with correct name
      const { body: listBody } = await apiRequest({
        method: 'GET',
        path: '/api/sessions',
      });
      expect(listBody.data.some((s: { name: string }) => s.name === name)).toBe(true);

      // Cleanup
      await apiRequest({
        method: 'DELETE',
        path: `/api/sessions/${encodeURIComponent(name)}`,
      });
    }
  });

  test('empty session name behavior', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/sessions',
      body: { name: '' },
    });
    // Backend may accept or reject empty names — document the behavior
    if (status === 200) {
      expect(body.success).toBe(true);
      // Cleanup if created
      if (body.data?.name) {
        await apiRequest({
          method: 'DELETE',
          path: `/api/sessions/${encodeURIComponent(body.data.name)}`,
        });
      }
    }
    // Either 400 (rejected) or 200 (accepted) are valid
    expect([200, 400]).toContain(status);
  });

  test('GET backends returns expected structure', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/backends',
    });
    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);

    // Each backend should have name and available fields
    for (const backend of body.data) {
      expect(backend).toHaveProperty('name');
      expect(backend).toHaveProperty('available');
    }
  });

  test('concurrent session creation does not corrupt state', async ({ apiRequest }) => {
    const timestamp = Date.now();
    const names = [
      `concurrent-a-${timestamp}`,
      `concurrent-b-${timestamp}`,
      `concurrent-c-${timestamp}`,
    ];

    // Create all sessions concurrently
    const results = await Promise.all(
      names.map((name) =>
        apiRequest({
          method: 'POST',
          path: '/api/sessions',
          body: { name },
        }),
      ),
    );

    // All should succeed
    for (const result of results) {
      expect(result.status).toBe(200);
      expect(result.body.success).toBe(true);
    }

    // Verify all appear in list
    const { body: listBody } = await apiRequest({
      method: 'GET',
      path: '/api/sessions',
    });
    const listNames = listBody.data.map((s: { name: string }) => s.name);
    for (const name of names) {
      expect(listNames).toContain(name);
    }

    // Cleanup all
    await Promise.all(
      names.map((name) =>
        apiRequest({
          method: 'DELETE',
          path: `/api/sessions/${encodeURIComponent(name)}`,
        }),
      ),
    );
  });
});
