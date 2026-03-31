import { expect, test } from './fixtures'

/**
 * Stress, concurrency, boundary, and edge-case E2E tests.
 *
 * These tests target in-memory CRUD endpoints that share a global MemoryStore
 * singleton. They are designed to surface real bugs: data races, index
 * corruption on snippet re-indexing, rate-limiting regressions, missing
 * validation, and inconsistent error envelopes.
 *
 * IMPORTANT: Tests that mutate state share the server process with other
 * test files. Use unique prefixes to avoid collisions.
 */

// biome-ignore lint/suspicious/noExplicitAny: Playwright fixture types are not easily importable
type ApiFn = (params: any) => Promise<{ status: number; body: any }>

const TS = () => `stress-${Date.now()}`

// ─── Helpers ────────────────────────────────────────────────────────────────

/** Create a profile via API, return its numeric ID. */
async function createProfile(apiRequest: ApiFn, overrides: Record<string, unknown> = {}) {
  const prefix = TS()
  const { status, body } = await apiRequest({
    method: 'POST',
    path: '/api/profiles',
    body: { profile_key: `pk-${prefix}`, name: `name-${prefix}`, sort_order: 0, ...overrides },
  })
  return { id: body.data?.id as number, status, body, prefix }
}

/** Create a snippet via API, return its index. */
async function createSnippet(apiRequest: ApiFn, overrides: Record<string, unknown> = {}) {
  const prefix = TS()
  const { status, body } = await apiRequest({
    method: 'POST',
    path: '/api/snippets',
    body: { name: `snip-${prefix}`, command: `echo ${prefix}`, ...overrides },
  })
  return { index: body.data?.index as number, status, body, prefix }
}

/** Create a role via API. */
async function createRole(apiRequest: ApiFn, overrides: Record<string, unknown> = {}) {
  const prefix = TS()
  const { status, body } = await apiRequest({
    method: 'POST',
    path: '/api/roles',
    body: { name: `role-${prefix}`, system_prompt: `prompt ${prefix}`, ...overrides },
  })
  return { id: body.data?.id as string, status, body, prefix }
}

/** Delete all created profiles by IDs. Ignores errors (may already be deleted). */
async function cleanupProfileIds(apiRequest: ApiFn, ids: number[]) {
  await Promise.allSettled(
    ids.map((id) =>
      apiRequest({ method: 'DELETE', path: `/api/profiles/${id}`, retryConfig: { maxRetries: 0 } }),
    ),
  )
}

/** Delete all created groups by IDs. Ignores errors. */
async function cleanupGroupIds(apiRequest: ApiFn, ids: number[]) {
  await Promise.allSettled(
    ids.map((id) =>
      apiRequest({ method: 'DELETE', path: `/api/groups/${id}`, retryConfig: { maxRetries: 0 } }),
    ),
  )
}

/** Delete all created roles by IDs. Ignores errors. */
async function cleanupRoleIds(apiRequest: ApiFn, ids: string[]) {
  await Promise.allSettled(
    ids.map((id) =>
      apiRequest({ method: 'DELETE', path: `/api/roles/${id}`, retryConfig: { maxRetries: 0 } }),
    ),
  )
}

// ─── Rapid Sequential Requests ──────────────────────────────────────────────

test.describe('Rapid sequential requests', () => {
  test('create 50 profiles in rapid succession — all should succeed and appear in list', async ({
    apiRequest,
  }) => {
    const createdIds: number[] = []

    // Create 50 profiles sequentially
    for (let i = 0; i < 50; i++) {
      const { id, status } = await createProfile(apiRequest)
      expect(status).toBe(200)
      expect(id).toBeDefined()
      expect(typeof id).toBe('number')
      createdIds.push(id)
    }

    // Verify all appear in list
    const { body: listBody } = await apiRequest({ method: 'GET', path: '/api/profiles' })
    expect(listBody.success).toBe(true)
    const profileIds = new Set((listBody.data as { id: number }[]).map((p) => p.id))
    for (const id of createdIds) {
      expect(profileIds.has(id)).toBe(true)
    }

    // Cleanup
    await cleanupProfileIds(apiRequest, createdIds)
  })

  test('create and delete 20 snippets in rapid succession — final list should have no leaks', async ({
    apiRequest,
  }) => {
    // Create 20 snippets
    const createdIndices: number[] = []
    for (let i = 0; i < 20; i++) {
      const { index, status } = await createSnippet(apiRequest)
      expect(status).toBe(200)
      expect(index).toBeDefined()
      createdIndices.push(index)
    }

    // Verify they exist
    const { body: beforeBody } = await apiRequest({ method: 'GET', path: '/api/snippets' })
    expect(beforeBody.success).toBe(true)
    const beforeCount = (beforeBody.data as unknown[]).length

    // Delete all snippets (delete from highest index first to avoid re-indexing confusion)
    for (let i = createdIndices.length - 1; i >= 0; i--) {
      const { status } = await apiRequest({
        method: 'DELETE',
        path: `/api/snippets/${createdIndices[i]}`,
        retryConfig: { maxRetries: 0 },
      })
      expect(status).toBe(200)
    }

    // Verify list length decreased by 20
    const { body: afterBody } = await apiRequest({ method: 'GET', path: '/api/snippets' })
    expect(afterBody.success).toBe(true)
    const afterCount = (afterBody.data as unknown[]).length
    expect(afterCount).toBe(beforeCount - 20)
  })

  test('rapid toggle pane status 10 times — final value should be last write', async ({
    apiRequest,
  }) => {
    const paneKey = `stress-pane-${Date.now()}`
    const statuses = ['idle', 'in_progress', 'done', 'failed', 'waiting']

    // Rapidly set status 10 times
    for (let i = 0; i < 10; i++) {
      const status = statuses[i % statuses.length]
      const { status: httpStatus } = await apiRequest({
        method: 'PUT',
        path: '/api/panes/status',
        body: { pane_key: paneKey, status },
      })
      expect(httpStatus).toBe(200)
    }

    // The final write was statuses[9 % 5] = statuses[4] = 'waiting'
    const lastStatus = statuses[9 % statuses.length]
    const { body } = await apiRequest({ method: 'GET', path: '/api/panes/status' })
    expect(body.success).toBe(true)

    const found = (body.data as { pane_key: string; status: string }[]).find(
      (s) => s.pane_key === paneKey,
    )
    expect(found).toBeDefined()
    expect(found.status).toBe(lastStatus)
  })
})

// ─── Concurrent-like Patterns (Promise.all) ─────────────────────────────────

test.describe('Concurrent-like patterns (Promise.all)', () => {
  test('10 simultaneous GET /api/profiles — all return consistent data', async ({ apiRequest }) => {
    const responses = await Promise.all(
      Array.from({ length: 10 }, () => apiRequest({ method: 'GET', path: '/api/profiles' })),
    )

    // All should succeed
    for (const res of responses) {
      expect(res.status).toBe(200)
      expect(res.body.success).toBe(true)
    }

    // All should return the same data (same profile list)
    const firstData = JSON.stringify(responses[0].body.data)
    for (let i = 1; i < responses.length; i++) {
      expect(JSON.stringify(responses[i].body.data)).toBe(firstData)
    }
  })

  test('5 simultaneous POST /api/groups — all succeed and all appear in list', async ({
    apiRequest,
  }) => {
    const timestamp = Date.now()
    const bodies = await Promise.all(
      Array.from({ length: 5 }, (_, i) =>
        apiRequest({
          method: 'POST',
          path: '/api/groups',
          body: { group_name: `concurrent-grp-${timestamp}-${i}`, sort_order: i },
        }),
      ),
    )

    // All should succeed
    const createdIds: number[] = []
    for (const res of bodies) {
      expect(res.status).toBe(200)
      expect(res.body.success).toBe(true)
      expect(res.body.data.id).toBeDefined()
      createdIds.push(res.body.data.id)
    }

    // Verify all appear in list
    const { body: listBody } = await apiRequest({ method: 'GET', path: '/api/groups' })
    expect(listBody.success).toBe(true)
    const listIds = new Set((listBody.data as { id: number }[]).map((g) => g.id))
    for (const id of createdIds) {
      expect(listIds.has(id)).toBe(true)
    }

    // Cleanup
    await cleanupGroupIds(apiRequest, createdIds)
  })

  test('mixed CRUD: create, read, update, delete simultaneously — no data corruption', async ({
    apiRequest,
  }) => {
    // Create a profile first
    const { id, prefix } = await createProfile(apiRequest)
    expect(id).toBeDefined()

    // Run concurrent operations: read, update, read, read
    const [readRes1, updateRes, readRes2, readRes3] = await Promise.all([
      apiRequest({ method: 'GET', path: `/api/profiles/${id}` }),
      apiRequest({
        method: 'PUT',
        path: `/api/profiles/${id}`,
        body: { profile_key: `pk-${prefix}`, name: `updated-${prefix}`, sort_order: 99 },
      }),
      apiRequest({ method: 'GET', path: `/api/profiles/${id}` }),
      apiRequest({ method: 'GET', path: '/api/profiles' }),
    ])

    // All reads should succeed
    expect(readRes1.status).toBe(200)
    expect(readRes2.status).toBe(200)
    expect(readRes3.status).toBe(200)

    // Update should succeed
    expect(updateRes.status).toBe(200)
    expect(updateRes.body.data.name).toBe(`updated-${prefix}`)
    expect(updateRes.body.data.sort_order).toBe(99)

    // Delete
    const { status: deleteStatus } = await apiRequest({
      method: 'DELETE',
      path: `/api/profiles/${id}`,
    })
    expect(deleteStatus).toBe(200)

    // Verify deleted
    const { status: getAfterDelete } = await apiRequest({
      method: 'GET',
      path: `/api/profiles/${id}`,
      retryConfig: { maxRetries: 0 },
    })
    expect(getAfterDelete).toBe(404)
  })
})

// ─── Large Payload Tests ────────────────────────────────────────────────────

test.describe('Large payload tests', () => {
  test('POST /api/snippets with 2000-char command — server should accept it', async ({
    apiRequest,
  }) => {
    const longCommand = 'echo '.repeat(500) // ~2500 chars
    const { status, body } = await createSnippet(apiRequest, { command: longCommand })

    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.command.length).toBe(longCommand.length)
  })

  test('POST /api/snippets with 10000-char name — server should accept or reject gracefully', async ({
    apiRequest,
  }) => {
    const longName = 'x'.repeat(10000)
    const { status, body } = await createSnippet(apiRequest, { name: longName })

    // Server should either accept (200) or reject with validation error (400/413)
    // but NEVER crash with 500
    expect([200, 400, 413]).toContain(status)
    if (status === 200) {
      expect(body.success).toBe(true)
    }
  })

  test('POST /api/profiles with very long name — server should accept or reject gracefully', async ({
    apiRequest,
  }) => {
    const longName = 'profile-'.repeat(2000) // ~14000 chars
    const { status, body } = await createProfile(apiRequest, { name: longName })

    // Should not 500
    expect(status).toBeLessThan(500)
    if (status === 200) {
      expect(body.success).toBe(true)
      // Cleanup
      await cleanupProfileIds(apiRequest, [body.data.id])
    }
  })

  test('POST /api/roles with very long system prompt — server should accept or reject gracefully', async ({
    apiRequest,
  }) => {
    const longPrompt = 'You are a helpful assistant. '.repeat(1000) // ~28000 chars
    const { status, body } = await createRole(apiRequest, { system_prompt: longPrompt })

    // Should not 500
    expect(status).toBeLessThan(500)
    if (status === 200) {
      expect(body.success).toBe(true)
      // Cleanup
      await cleanupRoleIds(apiRequest, [body.data.id])
    }
  })
})

// ─── Boundary Value Tests ───────────────────────────────────────────────────

test.describe('Boundary value tests', () => {
  test('POST /api/profiles with empty name — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/profiles',
      body: { profile_key: 'pk-empty-name', name: '', sort_order: 0 },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('name')
  })

  test('POST /api/profiles with empty profile_key — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/profiles',
      body: { profile_key: '', name: 'has-name', sort_order: 0 },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('profile_key')
  })

  test('POST /api/profiles with XSS characters in name — server should not crash', async ({
    apiRequest,
  }) => {
    const xssPayloads = [
      '<script>alert("xss")</script>',
      '"><img src=x onerror=alert(1)>',
      'javascript:alert(1)',
      '{{7*7}}',
      '${' + '7*7}',
    ]

    for (const payload of xssPayloads) {
      const { status, body } = await apiRequest({
        method: 'POST',
        path: '/api/profiles',
        body: {
          profile_key: `pk-xss-${Date.now()}-${Math.random().toString(36).slice(2)}`,
          name: payload,
          sort_order: 0,
        },
        retryConfig: { maxRetries: 0 },
      })

      // Should not 500 — either accept or reject
      expect(status).toBeLessThan(500)
      if (status === 200) {
        expect(body.success).toBe(true)
        // Cleanup
        await cleanupProfileIds(apiRequest, [body.data.id])
      }
    }
  })

  test('POST /api/snippets with empty command — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/snippets',
      body: { name: 'has-name', command: '' },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('command')
  })

  test('POST /api/snippets with empty name — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/snippets',
      body: { name: '', command: 'echo test' },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('name')
  })

  test('POST /api/roles with empty system_prompt — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/roles',
      body: { name: 'has-name', system_prompt: '' },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('system_prompt')
  })

  test('POST /api/roles with empty name — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/roles',
      body: { name: '', system_prompt: 'has prompt' },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('name')
  })

  test('POST /api/groups with empty group_name — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/groups',
      body: { group_name: '', sort_order: 0 },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('group_name')
  })

  test('POST /api/tasks/events with very large data object — should not crash', async ({
    apiRequest,
  }) => {
    // Build a large nested data object (~100KB)
    const largeData: Record<string, unknown> = {}
    for (let i = 0; i < 1000; i++) {
      largeData[`key_${i}`] = 'x'.repeat(100)
    }

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: {
        pane_key: `stress-pane-${Date.now()}`,
        event: 'test_event',
        data: largeData,
      },
      retryConfig: { maxRetries: 0 },
    })

    // Should not 500
    expect(status).toBeLessThan(500)
    if (status === 200) {
      expect(body.success).toBe(true)
    }
  })

  test('POST /api/tasks/events with empty pane_key — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: { pane_key: '', event: 'test' },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('pane_key')
  })

  test('POST /api/tasks/events with empty event — should return 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/tasks/events',
      body: { pane_key: 'test-pane', event: '' },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('event')
  })
})

// ─── Rate Limiting Tests ────────────────────────────────────────────────────

test.describe('Rate limiting tests', () => {
  test('rapid fire 30 requests may trigger 429 rate limiting (burst=20, 10/s)', async ({
    request,
  }) => {
    // Use raw Playwright request to avoid apiRequest's auto-retry on 5xx
    const responses = await Promise.all(
      Array.from({ length: 30 }, () =>
        request.fetch('/api/profiles').then((res) => ({ status: res.status() })),
      ),
    )

    // Count successes vs rate-limited
    const successes = responses.filter((r) => r.status === 200).length
    const rateLimited = responses.filter((r) => r.status === 429).length

    // Some should succeed, rate limiting may kick in after burst
    expect(successes).toBeGreaterThan(0)

    // If rate limiting kicked in, verify the response body is JSON with success:false
    if (rateLimited > 0) {
      const rateLimitRes = await request.fetch('/api/profiles')
      const text = await rateLimitRes.text()
      if (rateLimitRes.status() === 429) {
        const parsed = JSON.parse(text)
        expect(parsed.success).toBe(false)
        expect(parsed.error).toContain('rate limit')
      }
    }
  })

  test('GET /api/health under load — should always respond', async ({ request }) => {
    // Fire 20 health checks concurrently
    const start = Date.now()
    const responses = await Promise.all(
      Array.from({ length: 20 }, () => request.fetch('/api/health')),
    )

    const elapsed = Date.now() - start

    // All should return (some may be 429 from rate limiter, but should not hang)
    for (const res of responses) {
      expect([200, 429]).toContain(res.status())
    }

    // Should complete in reasonable time (< 5s)
    expect(elapsed).toBeLessThan(5000)
  })
})

// ─── ID / Path Traversal Tests ─────────────────────────────────────────────

test.describe('ID / path traversal tests', () => {
  test('GET /api/profiles/99999 — should return 404, not 500', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/profiles/99999',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
    expect(body.success).toBe(false)
    expect(body.error).toBeDefined()
  })

  test('GET /api/profiles/-1 — should return error, not 500', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/profiles/-1',
      retryConfig: { maxRetries: 0 },
    })

    // Negative IDs are out of range — should be 404, not 500
    expect(status).toBe(404)
  })

  test('GET /api/profiles/abc — should return 400 for invalid ID', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/profiles/abc',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('invalid')
  })

  test('DELETE /api/profiles/99999 — should return 404, not 500', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'DELETE',
      path: '/api/profiles/99999',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
    expect(body.success).toBe(false)
  })

  test('PUT /api/profiles/99999 — should return 404, not 500', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'PUT',
      path: '/api/profiles/99999',
      body: { profile_key: 'pk-dne', name: 'does-not-exist', sort_order: 0 },
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
    expect(body.success).toBe(false)
  })

  test('GET /api/snippets/99999 — should return 404, not 500', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/snippets/99999',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
    expect(body.success).toBe(false)
  })

  test('GET /api/snippets/abc — should return 400 for invalid index', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/snippets/abc',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('invalid')
  })

  test('GET /api/snippets/-1 — should return 404 for negative index', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/snippets/-1',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
  })

  test('DELETE /api/snippets/99999 — should return 404, not 500', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'DELETE',
      path: '/api/snippets/99999',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
    expect(body.success).toBe(false)
  })

  test('GET /api/roles/abc — should return 400 for invalid ID', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/roles/abc',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('invalid')
  })

  test('DELETE /api/roles/99999 — should return 404, not 500', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'DELETE',
      path: '/api/roles/99999',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(404)
    expect(body.success).toBe(false)
  })

  test('GET /api/groups/abc — should return 400 for invalid ID', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/groups/abc',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('invalid')
  })

  test('path traversal attempt /api/profiles/.. — should return error', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/profiles/..',
      retryConfig: { maxRetries: 0 },
    })

    // Should not succeed — either 400 (invalid ID) or some other error
    expect([400, 404]).toContain(status)
  })
})

// ─── Response Format Consistency ────────────────────────────────────────────

test.describe('Response format consistency', () => {
  test('error responses follow apiResponse envelope — success:false, error:string', async ({
    apiRequest,
  }) => {
    // Test various error-inducing requests
    const errorCases = [
      {
        method: 'POST' as const,
        path: '/api/profiles',
        body: { profile_key: 'pk', name: '' },
        expectedStatus: 400,
      },
      {
        method: 'POST' as const,
        path: '/api/snippets',
        body: { name: '', command: 'echo' },
        expectedStatus: 400,
      },
      {
        method: 'POST' as const,
        path: '/api/roles',
        body: { name: '', system_prompt: 'p' },
        expectedStatus: 400,
      },
      { method: 'GET' as const, path: '/api/profiles/abc', expectedStatus: 400 },
      { method: 'GET' as const, path: '/api/profiles/99999', expectedStatus: 404 },
      { method: 'DELETE' as const, path: '/api/profiles/99999', expectedStatus: 404 },
      {
        method: 'PUT' as const,
        path: '/api/profiles/99999',
        body: { profile_key: 'pk', name: 'n' },
        expectedStatus: 404,
      },
    ]

    for (const tc of errorCases) {
      const { status, body } = await apiRequest({
        method: tc.method,
        path: tc.path,
        body: tc.body,
        retryConfig: { maxRetries: 0 },
      })

      expect(status).toBe(tc.expectedStatus)

      // Verify envelope structure for apiResponse-based errors
      // NOTE: http.Error (used for 405) does NOT follow the envelope pattern
      // so we only check status codes that use writeAPIError
      if (status !== 405) {
        expect(body.success).toBe(false)
        expect(typeof body.error).toBe('string')
        expect(body.error.length).toBeGreaterThan(0)
      }
    }
  })

  test('success responses follow apiResponse envelope — success:true, data present', async ({
    apiRequest,
  }) => {
    const { status: listStatus, body: listBody } = await apiRequest({
      method: 'GET',
      path: '/api/profiles',
    })
    expect(listStatus).toBe(200)
    expect(listBody.success).toBe(true)
    expect('data' in listBody).toBe(true)

    const { status: snipStatus, body: snipBody } = await apiRequest({
      method: 'GET',
      path: '/api/snippets',
    })
    expect(snipStatus).toBe(200)
    expect(snipBody.success).toBe(true)
    expect('data' in snipBody).toBe(true)

    const { status: healthStatus, body: healthBody } = await apiRequest({
      method: 'GET',
      path: '/api/health',
    })
    expect(healthStatus).toBe(200)
    expect(healthBody.success).toBe(true)
    expect(healthBody.data.status).toBe('ok')

    const { status: backendsStatus, body: backendsBody } = await apiRequest({
      method: 'GET',
      path: '/api/backends',
    })
    expect(backendsStatus).toBe(200)
    expect(backendsBody.success).toBe(true)
    expect(Array.isArray(backendsBody.data)).toBe(true)
  })

  test('405 Method Not Allowed may not follow apiResponse envelope (known inconsistency)', async ({
    apiRequest,
  }) => {
    // These endpoints use http.Error instead of writeAPIError for 405
    // This test documents the inconsistency
    const { status: profilesStatus, body: profilesBody } = await apiRequest({
      method: 'PATCH',
      path: '/api/profiles',
      retryConfig: { maxRetries: 0 },
    })
    expect(profilesStatus).toBe(405)
    // http.Error returns plain text "Method not allowed", not JSON envelope
    // The apiRequest fixture may parse this as JSON and get body.success === undefined
    // This is a known inconsistency to document
    if (typeof profilesBody.success !== 'undefined') {
      expect(profilesBody.success).toBe(false)
    }

    const { status: tasksStatus } = await apiRequest({
      method: 'POST',
      path: '/api/tasks',
      retryConfig: { maxRetries: 0 },
    })
    expect(tasksStatus).toBe(405)
  })
})

// ─── Snippet Index Re-indexing Stress ───────────────────────────────────────

test.describe('Snippet index re-indexing under stress', () => {
  test('delete snippets from middle of list — remaining snippets get correct indices', async ({
    apiRequest,
  }) => {
    // Create 5 snippets
    const indices: number[] = []
    for (let i = 0; i < 5; i++) {
      const { index, status } = await createSnippet(apiRequest)
      expect(status).toBe(200)
      indices.push(index)
    }

    // Original: [0, 1, 2, 3, 4]
    // Delete index 2 (middle)
    const { status: deleteStatus } = await apiRequest({
      method: 'DELETE',
      path: `/api/snippets/${indices[2]}`,
      retryConfig: { maxRetries: 0 },
    })
    expect(deleteStatus).toBe(200)

    // After delete: [0, 1, 2, 3] (re-indexed)
    const { body } = await apiRequest({ method: 'GET', path: '/api/snippets' })
    const snippets = body.data as { index: number; name: string }[]
    expect(snippets.length).toBe(4)

    // Verify contiguous indices
    for (let i = 0; i < snippets.length; i++) {
      expect(snippets[i].index).toBe(i)
    }

    // Cleanup remaining snippets (delete from highest index first)
    for (let i = snippets.length - 1; i >= 0; i--) {
      await apiRequest({
        method: 'DELETE',
        path: `/api/snippets/${snippets[i].index}`,
        retryConfig: { maxRetries: 0 },
      })
    }
  })

  test('interleaved create and delete — no index corruption', async ({ apiRequest }) => {
    // Create, delete, create, delete, create — all targeting index 0
    for (let round = 0; round < 5; round++) {
      const { status: createStatus, index } = await createSnippet(apiRequest)
      expect(createStatus).toBe(200)
      expect(index).toBe(0) // Always 0 since we delete each time

      const { status: deleteStatus } = await apiRequest({
        method: 'DELETE',
        path: '/api/snippets/0',
        retryConfig: { maxRetries: 0 },
      })
      expect(deleteStatus).toBe(200)
    }

    // Verify the store is still healthy by creating one more and deleting it
    const { status: finalCreateStatus, index: finalIndex } = await createSnippet(apiRequest)
    expect(finalCreateStatus).toBe(200)
    expect(finalIndex).toBe(0)

    await apiRequest({
      method: 'DELETE',
      path: '/api/snippets/0',
      retryConfig: { maxRetries: 0 },
    })
  })
})

// ─── Invalid JSON Body ──────────────────────────────────────────────────────

test.describe('Invalid JSON body handling', () => {
  test('POST /api/profiles with invalid JSON — returns 400', async ({ request }) => {
    const res = await request.fetch('/api/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: '{invalid json',
    })

    expect(res.status()).toBe(400)
  })

  test('POST /api/snippets with non-JSON content type — returns 400', async ({ request }) => {
    const res = await request.fetch('/api/snippets', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      data: 'name=test&command=echo',
    })

    expect(res.status()).toBe(400)
  })

  test('PUT /api/panes/status with invalid JSON — returns 400', async ({ request }) => {
    const res = await request.fetch('/api/panes/status', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json at all',
    })

    expect(res.status()).toBe(400)
  })

  test('POST /api/tasks/events with invalid JSON — returns 400', async ({ request }) => {
    const res = await request.fetch('/api/tasks/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'broken json{{{',
    })

    expect(res.status()).toBe(400)
  })
})

// ─── Concurrent Pane Status Updates ─────────────────────────────────────────

test.describe('Concurrent pane status updates', () => {
  test('simultaneous PUT /api/panes/status for different panes — all succeed', async ({
    apiRequest,
  }) => {
    const paneCount = 20
    const paneKeys = Array.from(
      { length: paneCount },
      (_, i) => `concurrent-pane-${Date.now()}-${i}`,
    )
    const statuses = ['idle', 'in_progress', 'done', 'failed', 'waiting']

    // Update all panes concurrently
    const results = await Promise.all(
      paneKeys.map((key) =>
        apiRequest({
          method: 'PUT',
          path: '/api/panes/status',
          body: { pane_key: key, status: statuses[Math.floor(Math.random() * statuses.length)] },
        }),
      ),
    )

    // All should succeed
    for (const res of results) {
      expect(res.status).toBe(200)
      expect(res.body.success).toBe(true)
    }

    // Verify all appear in GET
    const { body } = await apiRequest({ method: 'GET', path: '/api/panes/status' })
    expect(body.success).toBe(true)
    const statusMap = new Map(
      (body.data as { pane_key: string; status: string }[]).map((s) => [s.pane_key, s.status]),
    )
    for (const key of paneKeys) {
      expect(statusMap.has(key)).toBe(true)
    }
  })
})

// ─── Edge Cases with Numeric Values ─────────────────────────────────────────

test.describe('Numeric boundary values', () => {
  test('GET /api/tasks with page=-1 and limit=-1 — should not crash', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/tasks?page=-1&limit=-1',
      retryConfig: { maxRetries: 0 },
    })

    // Should return 200 with some data (even if empty)
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data).toBeDefined()
  })

  test('GET /api/tasks with very large page and limit — should not crash', async ({
    apiRequest,
  }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/tasks?page=999999&limit=999999',
      retryConfig: { maxRetries: 0 },
    })

    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data).toBeDefined()
  })

  test('GET /api/tasks with non-numeric page and limit — should not crash', async ({
    apiRequest,
  }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/tasks?page=abc&limit=xyz',
      retryConfig: { maxRetries: 0 },
    })

    // Atoi fails → page=0, limit=0, should still return 200
    expect(status).toBe(200)
    expect(body.success).toBe(true)
  })
})

// ─── Method Not Allowed Consistency ─────────────────────────────────────────

test.describe('HTTP method enforcement', () => {
  test('DELETE on collection endpoints returns 405', async ({ apiRequest }) => {
    const endpoints = ['/api/profiles', '/api/snippets', '/api/groups', '/api/roles', '/api/tasks']

    for (const path of endpoints) {
      const { status } = await apiRequest({
        method: 'DELETE',
        path,
        retryConfig: { maxRetries: 0 },
      })
      expect(status).toBe(405)
    }
  })

  test('POST on read-only endpoints returns 405 or error', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'POST',
      path: '/api/backends',
      retryConfig: { maxRetries: 0 },
    })
    expect([405, 400]).toContain(status)
  })
})
