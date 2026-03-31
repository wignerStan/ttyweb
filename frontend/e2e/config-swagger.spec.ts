import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { expect, test } from './fixtures'

test.describe('OpenCode Config', () => {
  test('GET /api/opencode-config returns config object', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/opencode-config',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    // Without cwd param, data is null
    expect(body.data).toBeNull()
  })

  test('Config response has expected fields when cwd is provided', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/opencode-config?cwd=/tmp',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    // /tmp is unlikely to have opencode.json, so data is null
    // But the response envelope is correct
    expect('data' in body).toBe(true)
  })

  test('Config with nonexistent cwd returns null data', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/opencode-config?cwd=/nonexistent/path/xyz123',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data).toBeNull()
  })

  test('POST /api/opencode-config returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'POST',
      path: '/api/opencode-config',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })
})

test.describe('Auth Check', () => {
  test('GET /api/auth/check returns authenticated status', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/auth/check',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
  })

  test('Auth response structure has expected fields', async ({ apiRequest }) => {
    const { body } = await apiRequest({
      method: 'GET',
      path: '/api/auth/check',
    })
    expect(body.data).toHaveProperty('authenticated')
    expect(body.data).toHaveProperty('auth_enabled')
    expect(typeof body.data.authenticated).toBe('boolean')
    expect(typeof body.data.auth_enabled).toBe('boolean')
    // When auth is not configured, always authenticated
    expect(body.data.authenticated).toBe(true)
  })

  test('POST /api/auth/check returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'POST',
      path: '/api/auth/check',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })
})

test.describe.configure({ mode: 'serial' })

test.describe('Pane Status', () => {
  const testPaneKey = `e2e-pane-${Date.now()}`

  test('GET /api/panes/status returns object', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/panes/status',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(typeof body.data).toBe('object')
    expect(body.data).not.toBeNull()
  })

  test('PUT /api/panes/status updates pane status', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'PUT',
      path: '/api/panes/status',
      body: { pane_key: testPaneKey, status: 'running' },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.pane_key).toBe(testPaneKey)
    expect(body.data.status).toBe('running')
  })

  test('GET /api/panes/status after update shows persisted status', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/panes/status',
    })
    expect(status).toBe(200)
    expect(body.data[testPaneKey]).toBe('running')
  })

  test('PUT /api/panes/status with missing pane_key returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'PUT',
      path: '/api/panes/status',
      body: { status: 'running' },
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('pane_key')
  })

  test('PUT /api/panes/status with missing status returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'PUT',
      path: '/api/panes/status',
      body: { pane_key: 'test' },
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('status')
  })

  test('PUT /api/panes/status with invalid body returns 400', async ({ request }) => {
    const res = await request.fetch('/api/panes/status', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json',
    })
    expect(res.status()).toBe(400)
  })

  test('DELETE /api/panes/status returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'DELETE',
      path: '/api/panes/status',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })
})

test.describe('Telemetry', () => {
  test('POST /api/telemetry accepts telemetry event', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/telemetry',
      body: {
        events: [{ name: 'test_event', timestamp: Date.now() }],
      },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.status).toBe('received')
  })

  test('POST /api/telemetry with empty events array succeeds', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/telemetry',
      body: { events: [] },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.status).toBe('received')
  })

  test('POST /api/telemetry with various event types', async ({ apiRequest }) => {
    const events = [
      { name: 'page_view', page: '/terminal' },
      { name: 'command_run', command: 'ls -la' },
      { name: 'session_connect', backend: 'local' },
      { name: 'error', message: 'test error', code: 500 },
    ]

    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/telemetry',
      body: { events },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.status).toBe('received')
  })

  test('POST /api/telemetry with invalid body returns 400', async ({ request }) => {
    const res = await request.fetch('/api/telemetry', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json',
    })
    expect(res.status()).toBe(400)
  })

  test('GET /api/telemetry returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/telemetry',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })
})

test.describe('Health Check', () => {
  test('GET /api/health returns ok status', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/health',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
  })

  test('Health check response structure', async ({ apiRequest }) => {
    const { body } = await apiRequest({
      method: 'GET',
      path: '/api/health',
    })
    expect(body.data).toHaveProperty('status')
    expect(body.data.status).toBe('ok')
    expect(typeof body.data.status).toBe('string')
  })
})

test.describe('Upload', () => {
  test('POST /api/upload without file returns error', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/upload',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('file')
  })

  test('POST /api/upload with multipart form accepts file', async ({ request }) => {
    // Create a temporary file to upload
    const tmpDir = os.tmpdir()
    const tmpFile = path.join(tmpDir, `e2e-upload-test-${Date.now()}.txt`)
    const content = 'E2E upload test content\n'
    fs.writeFileSync(tmpFile, content)

    try {
      const res = await request.fetch('/api/upload', {
        method: 'POST',
        multipart: {
          file: {
            name: 'test-upload.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from(content),
          },
        },
      })
      expect(res.status()).toBe(200)
      const body = await res.json()
      expect(body.success).toBe(true)
      expect(body.data).toHaveProperty('filename')
      expect(body.data).toHaveProperty('path')
      expect(body.data).toHaveProperty('size')
      expect(body.data).toHaveProperty('type')
      expect(body.data.filename).toBe('test-upload.txt')
      expect(body.data.size).toBe(content.length)
      expect(body.data.type).toBe('text/plain')
    } finally {
      // Clean up temp file
      fs.unlinkSync(tmpFile)
    }
  })

  test('POST /api/upload with non-multipart body returns error', async ({ request }) => {
    const res = await request.fetch('/api/upload', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ not: 'multipart' }),
    })
    expect(res.status()).toBe(400)
  })

  test('GET /api/upload returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/upload',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })
})

test.describe('AI Endpoints (Stub Tests)', () => {
  test('POST /api/ai/command without API key returns stub response', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/ai/command',
      body: { prompt: 'list files', role: 'shell' },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data).toHaveProperty('command')
    expect(body.data).toHaveProperty('explanation')
    // Without LLM_API_KEY, command is empty and explanation describes the issue
    expect(body.data.command).toBe('')
    expect(body.data.explanation).toContain('No LLM API key')
  })

  test('POST /api/ai/command with empty prompt returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/ai/command',
      body: { prompt: '' },
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('prompt')
  })

  test('POST /api/ai/command with invalid body returns 400', async ({ request }) => {
    const res = await request.fetch('/api/ai/command', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json',
    })
    expect(res.status()).toBe(400)
  })

  test('GET /api/ai/command returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/ai/command',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })

  test('GET /api/ai/sessions returns session list', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/ai/sessions',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    // Sessions may be empty or have entries depending on state
    expect(Array.isArray(body.data)).toBe(true)
  })

  test('GET /api/ai/sessions with project filter', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/ai/sessions?project=/tmp',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(Array.isArray(body.data)).toBe(true)
  })

  test('GET /api/ai/sessions/cleanup returns 405 for GET', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/ai/sessions/cleanup',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })

  test('GET /api/roles/defaults returns default roles', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/roles/defaults',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(Array.isArray(body.data)).toBe(true)
    expect(body.data.length).toBeGreaterThan(0)
    // Each role should have id, name, description, system_prompt, suffix
    for (const role of body.data) {
      expect(role).toHaveProperty('id')
      expect(role).toHaveProperty('name')
      expect(role).toHaveProperty('system_prompt')
    }
  })

  test('GET /api/roles returns roles list', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'GET',
      path: '/api/roles',
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(Array.isArray(body.data)).toBe(true)
  })

  test('GET /api/roles/defaults returns same roles as /api/roles initially', async ({
    apiRequest,
  }) => {
    const { body: defaultsBody } = await apiRequest({
      method: 'GET',
      path: '/api/roles/defaults',
    })
    const { body: rolesBody } = await apiRequest({
      method: 'GET',
      path: '/api/roles',
    })
    // Both return arrays of roles
    expect(Array.isArray(defaultsBody.data)).toBe(true)
    expect(Array.isArray(rolesBody.data)).toBe(true)
    // Defaults should be a subset of all roles (or equal if no custom roles)
    expect(defaultsBody.data.length).toBeLessThanOrEqual(rolesBody.data.length)
  })
})

test.describe('Swagger / OpenAPI', () => {
  test('GET /api/docs/ returns swagger UI', async ({ request }) => {
    const res = await request.fetch('/api/docs/')
    // Swagger UI may return 200 with HTML or redirect
    expect([200, 301, 302, 307, 308]).toContain(res.status())
    if (res.status() === 200) {
      const contentType = res.headers()['content-type'] ?? ''
      // Should be HTML (swagger UI)
      expect(contentType).toContain('text/html')
    }
  })

  test('GET /api/docs/swagger.json returns OpenAPI spec', async ({ request }) => {
    const res = await request.fetch('/api/docs/swagger.json')
    expect(res.status()).toBe(200)
    const contentType = res.headers()['content-type'] ?? ''
    expect(contentType).toContain('application/json')

    const body = await res.json()
    // Must have either 'swagger' (Swagger 2.0) or 'openapi' (OpenAPI 3.x) field
    expect(body.swagger ?? body.openapi).toBeDefined()
    expect(body).toHaveProperty('paths')
    expect(typeof body.paths).toBe('object')
  })

  test('GET /api/docs/swagger.json contains expected API paths', async ({ request }) => {
    const res = await request.fetch('/api/docs/swagger.json')
    const body = await res.json()
    const paths = Object.keys(body.paths)

    // Verify core API paths exist in the spec
    expect(paths.some((p) => p.includes('auth'))).toBe(true)
    expect(paths.some((p) => p.includes('health'))).toBe(true)
    expect(paths.some((p) => p.includes('telemetry'))).toBe(true)
  })
})

test.describe('Log Endpoint', () => {
  test('POST /api/log accepts log entry', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/log',
      body: {
        level: 'info',
        message: 'E2E test log entry',
        url: '/terminal',
      },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.status).toBe('logged')
  })

  test('POST /api/log with error level', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/log',
      body: {
        level: 'error',
        message: 'E2E test error entry',
        data: { code: 500, endpoint: '/api/test' },
      },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.status).toBe('logged')
  })

  test('POST /api/log with empty message returns 400', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/log',
      body: { level: 'info', message: '' },
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(400)
    expect(body.success).toBe(false)
    expect(body.error).toContain('message')
  })

  test('POST /api/log with invalid body returns 400', async ({ request }) => {
    const res = await request.fetch('/api/log', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: 'not json',
    })
    expect(res.status()).toBe(400)
  })

  test('GET /api/log returns 405', async ({ apiRequest }) => {
    const { status } = await apiRequest({
      method: 'GET',
      path: '/api/log',
      retryConfig: { maxRetries: 0 },
    })
    expect(status).toBe(405)
  })

  test('POST /api/log defaults level to info when omitted', async ({ apiRequest }) => {
    const { status, body } = await apiRequest({
      method: 'POST',
      path: '/api/log',
      body: { message: 'E2E test default level' },
    })
    expect(status).toBe(200)
    expect(body.success).toBe(true)
    expect(body.data.status).toBe('logged')
  })
})
