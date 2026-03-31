import { test, expect } from './fixtures';

test.describe('Profile persistence', () => {
  test('POST /api/profiles creates a profile', async ({ apiRequest }) => {
    const profileKey = `pw-profile-${Date.now()}`;
    const name = `Profile ${Date.now()}`;

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/profiles',
      body: { profile_key: profileKey, name, sort_order: 10 },
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.profile_key).toBe(profileKey);
    expect(body.data.name).toBe(name);
    expect(body.data.sort_order).toBe(10);
    expect(typeof body.data.id).toBe('number');

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/profiles/${body.data.id}`,
    });
  });

  test('GET /api/profiles lists profiles as array', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/profiles',
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('PUT /api/profiles/{id} updates a profile', async ({ apiRequest }) => {
    const profileKey = `pw-update-${Date.now()}`;
    const originalName = `Original ${Date.now()}`;
    const updatedName = `Updated ${Date.now()}`;

    // Create
    const { body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/profiles',
      body: { profile_key: profileKey, name: originalName, sort_order: 1 },
    });
    const profileId = createBody.data.id;

    // Update
    const { status, body: updateBody } = await apiRequest({
      method: 'PUT',
      path: `/api/profiles/${profileId}`,
      body: { profile_key: profileKey, name: updatedName, sort_order: 99 },
    });

    expect(status).toBe(200);
    expect(updateBody.success).toBe(true);
    expect(updateBody.data.id).toBe(profileId);
    expect(updateBody.data.name).toBe(updatedName);
    expect(updateBody.data.sort_order).toBe(99);

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/profiles/${profileId}`,
    });
  });

  test('DELETE /api/profiles/{id} removes a profile', async ({ apiRequest }) => {
    const profileKey = `pw-delete-${Date.now()}`;
    const name = `Delete Me ${Date.now()}`;

    // Create
    const { body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/profiles',
      body: { profile_key: profileKey, name },
    });
    const profileId = createBody.data.id;

    // Verify it exists in list
    const { body: listBefore } = await apiRequest({
      method: 'GET',
      path: '/api/profiles',
    });
    expect(listBefore.data.some((p: { id: number }) => p.id === profileId)).toBe(true);

    // Delete
    const { status, body: deleteBody } = await apiRequest({
      method: 'DELETE',
      path: `/api/profiles/${profileId}`,
    });
    expect(status).toBe(200);
    expect(deleteBody.success).toBe(true);
    expect(deleteBody.data.status).toBe('deleted');

    // Verify gone from list
    const { body: listAfter } = await apiRequest({
      method: 'GET',
      path: '/api/profiles',
    });
    expect(listAfter.data.some((p: { id: number }) => p.id === profileId)).toBe(false);
  });

  test('POST /api/profiles with missing fields returns error', async ({ request }) => {
    // Missing profile_key
    const res1 = await request.fetch('/api/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ name: 'no key' }),
    });
    const body1 = await res1.json();
    expect(res1.status()).toBe(400);
    expect(body1.success).toBe(false);
    expect(body1.error).toContain('profile_key');

    // Missing name
    const res2 = await request.fetch('/api/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ profile_key: 'no-name' }),
    });
    const body2 = await res2.json();
    expect(res2.status()).toBe(400);
    expect(body2.success).toBe(false);
    expect(body2.error).toContain('name');

    // Invalid JSON
    const res3 = await request.fetch('/api/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json',
    });
    const body3 = await res3.json();
    expect(res3.status()).toBe(400);
    expect(body3.success).toBe(false);
  });
});

test.describe('Group persistence', () => {
  test('POST /api/groups creates a group', async ({ apiRequest }) => {
    const groupName = `pw-group-${Date.now()}`;
    const profileKey = `pw-gp-${Date.now()}`;

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/groups',
      body: { group_name: groupName, profile_key: profileKey, sort_order: 5 },
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.group_name).toBe(groupName);
    expect(body.data.profile_key).toBe(profileKey);
    expect(body.data.sort_order).toBe(5);
    expect(typeof body.data.id).toBe('number');

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/groups/${body.data.id}`,
    });
  });

  test('GET /api/groups?profile_key=X filters by profile', async ({ apiRequest }) => {
    const profileKey = `pw-filter-${Date.now()}`;
    const groupName = `FilteredGroup ${Date.now()}`;

    // Create a group with a specific profile_key
    await apiRequest({
      method: 'POST',
      path: '/api/groups',
      body: { group_name: groupName, profile_key: profileKey },
    });

    // Filter by that profile_key
    const { status, body } = await apiRequest({
      method: 'GET',
      path: `/api/groups?profile_key=${encodeURIComponent(profileKey)}`,
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data.some((g: { group_name: string }) => g.group_name === groupName)).toBe(true);

    // Filter by a non-existent profile_key should return empty
    const { body: emptyBody } = await apiRequest({
      method: 'GET',
      path: '/api/groups?profile_key=nonexistent-xyz-999',
    });
    expect(emptyBody.data).toHaveLength(0);
  });

  test('full group lifecycle: create, update, delete', async ({ apiRequest }) => {
    const originalName = `Lifecycle ${Date.now()}`;
    const updatedName = `Updated Lifecycle ${Date.now()}`;
    const profileKey = `pw-lifecycle-${Date.now()}`;

    // Create
    const { status: createStatus, body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/groups',
      body: { group_name: originalName, profile_key: profileKey, sort_order: 1 },
    });
    expect(createStatus).toBe(200);
    expect(createBody.success).toBe(true);
    const groupId = createBody.data.id;

    // Update
    const { status: updateStatus, body: updateBody } = await apiRequest({
      method: 'PUT',
      path: `/api/groups/${groupId}`,
      body: { group_name: updatedName, profile_key: profileKey, sort_order: 100 },
    });
    expect(updateStatus).toBe(200);
    expect(updateBody.data.group_name).toBe(updatedName);
    expect(updateBody.data.sort_order).toBe(100);

    // Delete
    const { status: deleteStatus, body: deleteBody } = await apiRequest({
      method: 'DELETE',
      path: `/api/groups/${groupId}`,
    });
    expect(deleteStatus).toBe(200);
    expect(deleteBody.data.status).toBe('deleted');

    // Verify deleted
    const { body: listBody } = await apiRequest({
      method: 'GET',
      path: `/api/groups?profile_key=${encodeURIComponent(profileKey)}`,
    });
    expect(listBody.data.some((g: { id: number }) => g.id === groupId)).toBe(false);
  });
});

test.describe('Snippet persistence', () => {
  test('POST /api/snippets creates a snippet', async ({ apiRequest }) => {
    const name = `pw-snippet-${Date.now()}`;
    const command = `echo ${Date.now()}`;

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/snippets',
      body: { name, command },
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.name).toBe(name);
    expect(body.data.command).toBe(command);
    expect(typeof body.data.index).toBe('number');

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/snippets/${body.data.index}`,
    });
  });

  test('full snippet lifecycle: create, update, delete', async ({ apiRequest }) => {
    const originalName = `SnippetOrig ${Date.now()}`;
    const updatedName = `SnippetUpdated ${Date.now()}`;
    const originalCmd = `echo original-${Date.now()}`;
    const updatedCmd = `echo updated-${Date.now()}`;

    // Create
    const { status: createStatus, body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/snippets',
      body: { name: originalName, command: originalCmd },
    });
    expect(createStatus).toBe(200);
    expect(createBody.success).toBe(true);
    const snippetIndex = createBody.data.index;

    // Update
    const { status: updateStatus, body: updateBody } = await apiRequest({
      method: 'PUT',
      path: `/api/snippets/${snippetIndex}`,
      body: { name: updatedName, command: updatedCmd },
    });
    expect(updateStatus).toBe(200);
    expect(updateBody.data.name).toBe(updatedName);
    expect(updateBody.data.command).toBe(updatedCmd);

    // Delete
    const { status: deleteStatus, body: deleteBody } = await apiRequest({
      method: 'DELETE',
      path: `/api/snippets/${snippetIndex}`,
    });
    expect(deleteStatus).toBe(200);
    expect(deleteBody.data.status).toBe('deleted');

    // Verify gone from list
    const { body: listBody } = await apiRequest({
      method: 'GET',
      path: '/api/snippets',
    });
    expect(listBody.data.some((s: { name: string }) => s.name === updatedName)).toBe(false);
  });
});

test.describe('Task events', () => {
  test('POST /api/tasks/events creates a task event', async ({ apiRequest }) => {
    const paneKey = `pw-pane-${Date.now()}`;

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: {
        pane_key: paneKey,
        event: 'conversation',
        task_id: `task-${Date.now()}`,
        data: { message: 'hello' },
      },
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.pane_key).toBe(paneKey);
    expect(body.data.event).toBe('conversation');
    expect(typeof body.data.id).toBe('number');
  });

  test('GET /api/tasks lists task events with total count', async ({ apiRequest }) => {
    // Create a task event
    const paneKey = `pw-list-${Date.now()}`;
    await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: { pane_key: paneKey, event: 'test-list' },
    });

    // List tasks
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/tasks',
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(typeof body.data.total).toBe('number');
    expect(body.data.total).toBeGreaterThan(0);
    expect(Array.isArray(body.data.tasks)).toBe(true);
    expect(body.data.tasks.length).toBeGreaterThan(0);
  });

  test('POST /api/tasks/{id}/complete marks a task complete', async ({ apiRequest }) => {
    // Create a task event
    const paneKey = `pw-complete-${Date.now()}`;
    const { body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: { pane_key: paneKey, event: 'completion-test' },
    });
    const taskId = createBody.data.id;

    // Complete the task
    const { status, body } = await apiRequest({
      method: 'POST',
      path: `/api/tasks/${taskId}/complete`,
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.status).toBe('completed');

    // Verify it shows as completed in the list
    const { body: listBody } = await apiRequest({
      method: 'GET',
      path: '/api/tasks',
    });
    const completedTask = listBody.data.tasks.find((t: { id: number }) => t.id === taskId);
    expect(completedTask).toBeDefined();
    expect(completedTask.completed).toBe(true);
  });

  test('POST /api/tasks/events with missing fields returns error', async ({ request }) => {
    // Missing pane_key
    const res1 = await request.fetch('/api/tasks/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ event: 'test' }),
    });
    const body1 = await res1.json();
    expect(res1.status()).toBe(400);
    expect(body1.success).toBe(false);
    expect(body1.error).toContain('pane_key');

    // Missing event
    const res2 = await request.fetch('/api/tasks/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ pane_key: 'test' }),
    });
    const body2 = await res2.json();
    expect(res2.status()).toBe(400);
    expect(body2.success).toBe(false);
    expect(body2.error).toContain('event');
  });

  test('GET /api/tasks/events/{paneKey} filters events by pane', async ({ apiRequest }) => {
    const paneKey = `pw-filter-pane-${Date.now()}`;

    // Create events for this pane
    await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: { pane_key: paneKey, event: 'pane-filter-test' },
    });

    const { status, body } = await apiRequest({
      method: 'GET',
      path: `/api/tasks/events/${encodeURIComponent(paneKey)}`,
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data.length).toBeGreaterThan(0);
    for (const event of body.data) {
      expect(event.pane_key).toBe(paneKey);
    }
  });
});

test.describe('SSE event stream', () => {
  test('GET /api/tasks/events/stream returns correct headers', async ({ request }) => {
    const res = await request.fetch('/api/tasks/events/stream', {
      timeout: 3000,
    });

    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toContain('text/event-stream');
    expect(res.headers()['cache-control']).toContain('no-cache');
    expect(res.headers()['connection']).toContain('keep-alive');
  });

  test('GET /api/tasks/events/stream?paneKey=X returns correct headers', async ({ request }) => {
    const paneKey = `sse-pane-${Date.now()}`;
    const res = await request.fetch(`/api/tasks/events/stream?paneKey=${encodeURIComponent(paneKey)}`, {
      timeout: 3000,
    });

    expect(res.status()).toBe(200);
    expect(res.headers()['content-type']).toContain('text/event-stream');
  });

  test('POST /api/tasks/events/stream returns 405', async ({ request }) => {
    const res = await request.fetch('/api/tasks/events/stream', {
      method: 'POST',
    });
    expect(res.status()).toBe(405);
  });
});

test.describe('AI roles', () => {
  test('GET /api/roles lists builtin roles (7 expected)', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/roles',
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data.length).toBeGreaterThanOrEqual(7);

    // Verify known builtin roles are present
    const roleNames = body.data.map((r: { name: string }) => r.name);
    const expectedRoles = ['CLI', 'Operations', 'Frontend', 'Backend', 'Full-Stack', 'Security', 'DevOps'];
    for (const expected of expectedRoles) {
      expect(roleNames).toContain(expected);
    }
  });

  test('POST /api/roles creates a custom role', async ({ apiRequest }) => {
    const name = `CustomRole ${Date.now()}`;

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/roles',
      body: {
        name,
        description: 'Test custom role',
        system_prompt: 'You are a test assistant.',
      },
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(body.data.name).toBe(name);
    expect(body.data.description).toBe('Test custom role');
    expect(body.data.system_prompt).toBe('You are a test assistant.');
    expect(typeof body.data.id).toBe('number');

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/roles/${body.data.id}`,
    });
  });

  test('PUT /api/roles/{id} updates a role', async ({ apiRequest }) => {
    const originalName = `RoleBefore ${Date.now()}`;
    const updatedName = `RoleAfter ${Date.now()}`;

    // Create
    const { body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/roles',
      body: {
        name: originalName,
        description: 'before',
        system_prompt: 'Prompt before.',
      },
    });
    const roleId = createBody.data.id;

    // Update
    const { status, body: updateBody } = await apiRequest({
      method: 'PUT',
      path: `/api/roles/${roleId}`,
      body: {
        name: updatedName,
        description: 'after',
        system_prompt: 'Prompt after.',
      },
    });

    expect(status).toBe(200);
    expect(updateBody.success).toBe(true);
    expect(updateBody.data.id).toBe(roleId);
    expect(updateBody.data.name).toBe(updatedName);
    expect(updateBody.data.description).toBe('after');
    expect(updateBody.data.system_prompt).toBe('Prompt after.');

    // Cleanup
    await apiRequest({
      method: 'DELETE',
      path: `/api/roles/${roleId}`,
    });
  });

  test('DELETE /api/roles/{id} deletes a custom role', async ({ apiRequest }) => {
    const name = `DeleteRole ${Date.now()}`;

    // Create
    const { body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/roles',
      body: {
        name,
        description: 'to delete',
        system_prompt: 'Delete me.',
      },
    });
    const roleId = createBody.data.id;

    // Verify exists
    const { body: listBefore } = await apiRequest({
      method: 'GET',
      path: '/api/roles',
    });
    expect(listBefore.data.some((r: { id: number }) => r.id === roleId)).toBe(true);

    // Delete
    const { status, body: deleteBody } = await apiRequest({
      method: 'DELETE',
      path: `/api/roles/${roleId}`,
    });
    expect(status).toBe(200);
    expect(deleteBody.success).toBe(true);
    expect(deleteBody.data.status).toBe('deleted');

    // Verify gone
    const { body: listAfter } = await apiRequest({
      method: 'GET',
      path: '/api/roles',
    });
    expect(listAfter.data.some((r: { id: number }) => r.id === roleId)).toBe(false);
  });

  test('POST /api/roles with missing fields returns error', async ({ request }) => {
    // Missing name
    const res1 = await request.fetch('/api/roles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ system_prompt: 'test' }),
    });
    const body1 = await res1.json();
    expect(res1.status()).toBe(400);
    expect(body1.success).toBe(false);
    expect(body1.error).toContain('name');

    // Missing system_prompt
    const res2 = await request.fetch('/api/roles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ name: 'test' }),
    });
    const body2 = await res2.json();
    expect(res2.status()).toBe(400);
    expect(body2.success).toBe(false);
    expect(body2.error).toContain('system_prompt');
  });

  test('GET /api/roles/defaults returns builtin roles without custom roles', async ({ apiRequest }) => {
    // Create a custom role first
    const { body: createBody } = await apiRequest({
      method: 'POST',
      path: '/api/roles',
      body: {
        name: `DefaultsTest ${Date.now()}`,
        description: 'custom',
        system_prompt: 'custom prompt',
      },
    });

    // Get defaults — should always be exactly 7 builtin roles
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/roles/defaults',
    });

    expect(status).toBe(200);
    expect(body.success).toBe(true);
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data).toHaveLength(7);

    // Cleanup custom role
    await apiRequest({
      method: 'DELETE',
      path: `/api/roles/${createBody.data.id}`,
    });
  });
});
