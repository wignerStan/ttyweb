import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockFetchHttpError, mockFetchSuccess } from '../../../../test-helpers'
import { useInboxItems } from './useInboxItems'

vi.mock('../../../../utils/auth', () => ({
  getAuthHeader: () => '',
  getAuthHeaders: () => ({}),
}))

const items = [
  {
    id: '1',
    study_id: 's1',
    worker_id: 'w1',
    run_id: 'r1',
    kind: 'question',
    status: 'pending',
    title: 'Q',
    body: '',
    metadata: {},
    created_at: null,
    updated_at: null,
  },
]

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetchSuccess({ inbox_items: items }))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useInboxItems', () => {
  it('fetches inbox items on mount', async () => {
    const { result } = renderHook(() => useInboxItems())
    expect(fetch).toHaveBeenCalledWith('/api/butler/inbox_items', expect.any(Object))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.items).toEqual(items)
    expect(result.current.unreadCount).toBe(1)
  })

  it('passes filters to fetch URL', async () => {
    renderHook(() => useInboxItems({ study_id: 's1', status: 'pending' }))
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('study_id=s1'), expect.any(Object))
  })

  it('handles error', async () => {
    vi.stubGlobal('fetch', mockFetchHttpError(500))
    const { result } = renderHook(() => useInboxItems())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toBeTruthy()
  })
})
