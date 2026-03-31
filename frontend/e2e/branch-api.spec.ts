import { test, expect } from './fixtures'

test.describe('Branch API', () => {
  test('GET /api/branches without repo param returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/branches',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/repo.*required/i)
  })

  test('GET /api/branches?repo=nonexistent returns error', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/branches?repo=/nonexistent/path/to/repo',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/failed to list branches/i)
  })

  test('GET /api/branches?repo=/etc returns error (not a git repo)', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/branches?repo=/etc',
      retryConfig: { maxRetries: 0 },
    })
    // /etc is not a git repo, so worktree.ListBranches should fail
    expect(status).toBe(400)
    expect(body.success).toBe(false)
  })

  test('POST /api/branches with invalid JSON returns 400', async ({ request }) => {
    // Use raw Playwright request to send malformed JSON body
    const res = await request.fetch('/api/branches?repo=/tmp', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'not valid json{{{',
    })
    expect(res.status()).toBe(400)
    const body = await res.json()
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/invalid request body/i)
  })

  test('POST /api/branches with empty name returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/branches?repo=/tmp',
      body: { name: '' },
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/branch name is required/i)
  })

  test('DELETE /api/branches/invalid-branch?repo=nonexistent returns error', async ({
    apiRequest,
  }) => {
    const { status, body } = await apiRequest({
      method: 'DELETE',
      path: '/api/branches/invalid-branch-name-xyz?repo=/nonexistent/path',
      retryConfig: { maxRetries: 0 },
    })
    // The branch name contains hyphens which is valid per ValidateBranchName regex,
    // but the repo doesn't exist so the actual delete will fail
    expect(status).toBe(400)
    expect(body.success).toBe(false)
  })

  test('DELETE /api/branches with special chars in name returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'DELETE',
      path: '/api/branches/invalid%40name%21?repo=/tmp',
      retryConfig: { maxRetries: 0 },
    })
    // @ and ! are not valid in branch names per ValidateBranchName regex
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/invalid branch/i)
  })
})

test.describe('File Browser API', () => {
  test('GET /api/fs?path=/etc returns 403 (path not under allowed root)', async ({
    apiRequest,
  }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/fs?path=/etc',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(403)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/path not allowed/i)
  })

  test('GET /api/fs?path=nonexistent-dir returns 404', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/fs?path=nonexistent-directory-xyz-999',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(404)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/directory not found/i)
  })

  test('GET /api/fs with no path param returns CWD listing', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/fs',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(Array.isArray(body.data)).toBe(true)
  })

  test('POST /api/fs?path=/etc returns 405 method not allowed', async ({ request }) => {
    const res = await request.fetch('/api/fs?path=/etc', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: '{}',
    })
    expect(res.status()).toBe(405)
  })

  test('GET /api/fs?path=/proc/self returns error (forbidden or not found)', async ({
    apiRequest,
  }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/fs?path=/proc/self',
      retryConfig: { maxRetries: 0 },
    })
    // /proc/self is not under the allowed root, so should be 403
    expect(status).toBe(403)
    expect(body.success).toBe(false)
    expect(body.error).toMatch(/path not allowed/i)
  })

  test('GET /api/fs?path=. returns same listing as no path', async ({ apiRequest }) => {
    const { body: noPathBody } = await apiRequest({
      method: 'GET',
      path: '/api/fs',
    })
    const { body: dotPathBody } = await apiRequest({
      method: 'GET',
      path: '/api/fs?path=.',
    })
    expect(noPathBody.success).toBe(true)
    expect(dotPathBody.success).toBe(true)
    expect(dotPathBody.data).toEqual(noPathBody.data)
  })
})

test.describe('Version endpoint', () => {
  test('GET /api/version returns version info with expected fields', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/version',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data).toHaveProperty('version')
    expect(body.data).toHaveProperty('goVersion')
  })

  test('version response has non-empty version string', async ({ apiRequest }) => {
    const { body } = await apiRequest({
      method: 'GET',
      path: '/api/version',
    })
    expect(body.data.version).toBeTruthy()
    expect(typeof body.data.version).toBe('string')
    expect(body.data.version.length).toBeGreaterThan(0)
  })

  test('goVersion field contains a Go version pattern', async ({ apiRequest }) => {
    const { body } = await apiRequest({
      method: 'GET',
      path: '/api/version',
    })
    expect(body.data.goVersion).toBeTruthy()
    // Should match "go1.x.y" or "unknown" if build info unavailable
    expect(typeof body.data.goVersion).toBe('string')
    expect(body.data.goVersion.length).toBeGreaterThan(0)
  })
})
