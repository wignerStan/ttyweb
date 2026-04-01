import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock auth utility at module level
vi.mock('../utils/auth', () => ({
  getAuthHeader: vi.fn(() => null),
}))

import { getAuthHeader } from '../utils/auth'
import { useAuthFetch } from './useAuthFetch'

const mockGetAuthHeader = vi.mocked(getAuthHeader)

function getHeadersArg(mockFetch: ReturnType<typeof vi.fn>): Headers {
  return mockFetch.mock.calls[0]?.[1]?.headers as Headers
}

describe('useAuthFetch', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('includes auth header when credentials exist', async () => {
    mockGetAuthHeader.mockReturnValue('Basic dGVzdDp0ZXN0')

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: 'test' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAuthFetch())

    await act(async () => {
      await result.current.authFetch('/api/test')
    })

    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(getHeadersArg(mockFetch).get('Authorization')).toBe('Basic dGVzdDp0ZXN0')
  })

  it('omits auth header when credentials do not exist', async () => {
    mockGetAuthHeader.mockReturnValue(null)

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: 'test' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAuthFetch())

    await act(async () => {
      await result.current.authFetch('/api/test')
    })

    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(getHeadersArg(mockFetch).get('Authorization')).toBeNull()
  })

  it('passes through options including method and body', async () => {
    mockGetAuthHeader.mockReturnValue('Basic token')

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAuthFetch())

    await act(async () => {
      await result.current.authFetch('/api/data', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key: 'value' }),
      })
    })

    const callArgs = mockFetch.mock.calls[0]![1]!
    expect(callArgs.method).toBe('POST')
    expect(callArgs.body).toBe(JSON.stringify({ key: 'value' }))
    const headers = callArgs.headers as Headers
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('Authorization')).toBe('Basic token')
  })

  it('merges auth header with existing Content-Type header', async () => {
    mockGetAuthHeader.mockReturnValue('Bearer abc123')

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAuthFetch())

    await act(async () => {
      await result.current.authFetch('/api/items', {
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      })
    })

    const headers = getHeadersArg(mockFetch)
    expect(headers.get('Authorization')).toBe('Bearer abc123')
    expect(headers.get('Content-Type')).toBe('application/json')
    expect(headers.get('Accept')).toBe('application/json')
  })

  it('returns fetch response', async () => {
    mockGetAuthHeader.mockReturnValue(null)

    const responseData = { items: [1, 2, 3] }
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(responseData),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAuthFetch())

    let response: Response
    await act(async () => {
      response = await result.current.authFetch('/api/items')
    })

    expect(response!.ok).toBe(true)
    const data = await response!.json()
    expect(data).toEqual(responseData)
  })
})
