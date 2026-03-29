import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock auth utility at module level
vi.mock('../utils/auth', () => ({
  getAuthHeader: vi.fn(() => null),
}))

import { getAuthHeader } from '../utils/auth'
import { useNewWindow } from './useNewWindow'

const mockGetAuthHeader = vi.mocked(getAuthHeader)

describe('useNewWindow', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches quick dirs on mount', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ dirs: [{ name: 'home', path: '/home' }] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([{ name: 'home', path: '/home' }])
    })

    expect(mockFetch).toHaveBeenCalledWith('/api/tmux/quick-dirs', {
      headers: {},
    })
  })

  it('handles empty dirs from API', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ dirs: null }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })
  })

  it('handles fetch error for quick dirs', async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error('Network error'))
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })
  })

  it('sends auth header when fetching quick dirs', async () => {
    mockGetAuthHeader.mockReturnValue('Basic dGVzdDp0ZXN0')

    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ dirs: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/tmux/quick-dirs', {
        headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
      })
    })
  })

  it('creates a new window successfully', async () => {
    mockGetAuthHeader.mockReturnValue(null)
    const onSuccess = vi.fn()
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ window_id: 'w1' }),
      })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow(onSuccess))

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    await act(async () => {
      await result.current.createWindow('my-session', '/home', 'my-window')
    })

    expect(mockFetch).toHaveBeenLastCalledWith('/api/tmux/new-window', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session: 'my-session', dir: '/home', name: 'my-window' }),
    })
    expect(onSuccess).toHaveBeenCalled()
  })

  it('creates a new window without optional params', async () => {
    mockGetAuthHeader.mockReturnValue(null)
    const onSuccess = vi.fn()
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ window_id: 'w1' }),
      })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow(onSuccess))

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    await act(async () => {
      await result.current.createWindow('my-session')
    })

    expect(mockFetch).toHaveBeenLastCalledWith('/api/tmux/new-window', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session: 'my-session', dir: undefined, name: undefined }),
    })
  })

  it('handles error when creating window fails', async () => {
    mockGetAuthHeader.mockReturnValue(null)
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: 'session not found' }),
      })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    // The createWindow function throws, so we need to catch it
    let caughtError: Error | undefined
    await act(async () => {
      try {
        await result.current.createWindow('bad-session')
      } catch (err) {
        caughtError = err as Error
      }
    })

    expect(caughtError).toBeDefined()
    expect(caughtError!.message).toBe('session not found')

    // Error state should be set
    await waitFor(() => {
      expect(result.current.error).toBe('session not found')
    })
  })

  it('uses default error message when API returns no error field', async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({}),
      })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    await expect(
      act(async () => {
        await result.current.createWindow('bad-session')
      }),
    ).rejects.toThrow('Failed to create window')
  })

  it('sets loading during window creation', async () => {
    let resolveCreate: (value: unknown) => void
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveCreate = resolve
        }),
      )
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    // Start the create call (not awaited)
    const createPromise = result.current.createWindow('session')

    // loading should be true
    await waitFor(() => {
      expect(result.current.loading).toBe(true)
    })

    // Resolve the fetch
    await act(async () => {
      resolveCreate!({
        ok: true,
        json: () => Promise.resolve({ window_id: 'w1' }),
      })
      await createPromise
    })

    expect(result.current.loading).toBe(false)
  })

  it('resets loading and error on successful creation', async () => {
    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ window_id: 'w1' }),
      })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    await act(async () => {
      await result.current.createWindow('session')
    })

    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  it('sends auth header when creating window', async () => {
    mockGetAuthHeader.mockReturnValue('Basic dGVzdDp0ZXN0')

    const mockFetch = vi
      .fn()
      .mockResolvedValueOnce({
        json: () => Promise.resolve({ dirs: [] }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ window_id: 'w1' }),
      })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useNewWindow())

    await waitFor(() => {
      expect(result.current.quickDirs).toEqual([])
    })

    await act(async () => {
      await result.current.createWindow('session')
    })

    const createCall = mockFetch.mock.calls[1]!
    expect(createCall[1]!.headers.Authorization).toBe('Basic dGVzdDp0ZXN0')
  })
})
