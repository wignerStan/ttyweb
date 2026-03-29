import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock auth utility at module level
vi.mock('../utils/auth', () => ({
  getAuthHeader: vi.fn(() => null),
}))

import { getAuthHeader } from '../utils/auth'
import { useTmuxPrefix } from './useTmuxPrefix'

const mockGetAuthHeader = vi.mocked(getAuthHeader)

describe('useTmuxPrefix', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('returns default prefix (Ctrl+B) on mount', () => {
    const { result } = renderHook(() => useTmuxPrefix())

    expect(result.current.code).toBe('\x02')
    expect(result.current.label).toBe('Ctrl+B')
  })

  it('loads prefix from API and updates state', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: '\x01', label: 'Ctrl+A' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useTmuxPrefix())

    await waitFor(() => {
      expect(result.current.code).toBe('\x01')
      expect(result.current.label).toBe('Ctrl+A')
    })

    expect(mockFetch).toHaveBeenCalledWith('/api/tmux/config', {
      headers: undefined,
    })
  })

  it('sends auth header when available', async () => {
    mockGetAuthHeader.mockReturnValue('Basic dGVzdDp0ZXN0')

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: '\x01', label: 'Ctrl+A' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useTmuxPrefix())

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/tmux/config', {
        headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
      })
    })
  })

  it('keeps default prefix when API returns non-ok response', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve({ error: 'not found' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useTmuxPrefix())

    // Wait a bit for the fetch to resolve
    await act(async () => {
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1))
    })

    // Should still have default values
    expect(result.current.code).toBe('\x02')
    expect(result.current.label).toBe('Ctrl+B')
  })

  it('keeps default prefix when API returns data without code', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ other: 'data' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useTmuxPrefix())

    await act(async () => {
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1))
    })

    expect(result.current.code).toBe('\x02')
    expect(result.current.label).toBe('Ctrl+B')
  })

  it('uses "prefix" as label when API response has no label', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: '\x01' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useTmuxPrefix())

    await waitFor(() => {
      expect(result.current.code).toBe('\x01')
      expect(result.current.label).toBe('prefix')
    })
  })

  it('handles fetch error gracefully', async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error('Network error'))
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useTmuxPrefix())

    // Wait for the fetch to fail
    await act(async () => {
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1))
    })

    // Should still have default values (catch swallows the error)
    expect(result.current.code).toBe('\x02')
    expect(result.current.label).toBe('Ctrl+B')
  })
})
