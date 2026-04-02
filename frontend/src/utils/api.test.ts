import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiDelete, apiGet, apiPost, apiPut, authFetch } from './api'

vi.mock('./auth', () => ({
  getAuthHeader: vi.fn(() => 'Basic dGVzdDp0ZXN0'),
}))

describe('apiGet', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    globalThis.fetch = vi.fn()
  })

  it('calls fetch with auth header and returns parsed data', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true, data: [{ id: 1 }], error: undefined }),
    } as Response)

    const result = await apiGet<{ id: number }[]>('/api/sessions')

    expect(fetch).toHaveBeenCalledWith('/api/sessions', {
      headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
    })
    expect(result).toEqual([{ id: 1 }])
  })

  it('returns null when success is false', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: false, error: 'Not found' }),
    } as Response)

    const result = await apiGet('/api/sessions')
    expect(result).toBeNull()
  })

  it('returns null on network error', async () => {
    vi.mocked(globalThis.fetch).mockRejectedValueOnce(new Error('Network error'))

    const result = await apiGet('/api/sessions')
    expect(result).toBeNull()
  })
})

describe('apiPost', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    globalThis.fetch = vi.fn()
  })

  it('sends JSON body with Content-Type header', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true, data: { id: 42 } }),
    } as Response)

    const result = await apiPost<{ id: number }>('/api/sessions', { name: 'test' })

    expect(fetch).toHaveBeenCalledWith('/api/sessions', {
      method: 'POST',
      headers: {
        Authorization: 'Basic dGVzdDp0ZXN0',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name: 'test' }),
    })
    expect(result).toEqual({ id: 42 })
  })

  it('returns null on failure', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: false, error: 'bad request' }),
    } as Response)

    const result = await apiPost('/api/sessions', { name: 'test' })
    expect(result).toBeNull()
  })
})

describe('apiDelete', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    globalThis.fetch = vi.fn()
  })

  it('sends DELETE request', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true, data: null }),
    } as Response)

    await apiDelete('/api/sessions/test')

    expect(fetch).toHaveBeenCalledWith('/api/sessions/test', {
      method: 'DELETE',
      headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
    })
  })
})

describe('apiPut', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    globalThis.fetch = vi.fn()
  })

  it('sends PUT request with JSON body', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true, data: { updated: true } }),
    } as Response)

    const result = await apiPut<{ updated: boolean }>('/api/tasks/1', { status: 'done' })

    expect(fetch).toHaveBeenCalledWith('/api/tasks/1', {
      method: 'PUT',
      headers: {
        Authorization: 'Basic dGVzdDp0ZXN0',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ status: 'done' }),
    })
    expect(result).toEqual({ updated: true })
  })
})

describe('authFetch', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    globalThis.fetch = vi.fn()
  })

  it('adds auth headers to custom request', async () => {
    vi.mocked(globalThis.fetch).mockResolvedValueOnce({
      ok: true,
      body: null,
    } as Response)

    await authFetch('/api/custom', { method: 'POST', headers: { 'X-Custom': 'value' } })

    expect(fetch).toHaveBeenCalledWith('/api/custom', {
      method: 'POST',
      headers: { Authorization: 'Basic dGVzdDp0ZXN0', 'X-Custom': 'value' },
    })
  })
})
