import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock auth utility at module level
vi.mock('../utils/auth', () => ({
  getAuthHeader: vi.fn(() => null),
}))

import { getAuthHeader } from '../utils/auth'
import { useAuthFetch } from './useAuthFetch'

const mockGetAuthHeader = vi.mocked(getAuthHeader)

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

    expect(mockFetch).toHaveBeenCalledWith('/api/test', {
      headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
    })
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

    expect(mockFetch).toHaveBeenCalledWith('/api/test', {
      headers: {},
    })
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
        headers: { 'Content-Type': 'application/json' } as Record<string, string>,
        body: JSON.stringify({ key: 'value' }),
      })
    })

    expect(mockFetch).toHaveBeenCalledWith('/api/data', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Basic token',
      },
      body: JSON.stringify({ key: 'value' }),
    })
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

// Need act for async operations in tests
import { act } from '@testing-library/react'
