import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockFetchHttpError, mockFetchSuccess } from '../../../../test-helpers'
import { useActivityEvents } from './useActivityEvents'

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

const events = [
  {
    id: '1',
    study_id: 's1',
    worker_id: 'w1',
    event_type: 'task_started',
    summary: 'Started',
    detail: '',
    created_at: null,
  },
]

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetchSuccess({ activity_events: events }))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useActivityEvents', () => {
  it('fetches events on mount with correct params', async () => {
    const { result } = renderHook(() => useActivityEvents('s1'))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/butler/activity_events?limit=20&study_id=s1'),
      expect.any(Object),
    )
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.events).toEqual(events)
  })

  it('updates events on successful fetch', async () => {
    const { result } = renderHook(() => useActivityEvents())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.events).toEqual(events)
  })

  it('handles error state', async () => {
    vi.stubGlobal('fetch', mockFetchHttpError(500))
    const { result } = renderHook(() => useActivityEvents())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toBeTruthy()
  })

  it('refetch reloads events', async () => {
    const { result } = renderHook(() => useActivityEvents())
    await waitFor(() => expect(result.current.loading).toBe(false))
    const newEvents = [{ ...events[0], summary: 'Updated' }]
    vi.stubGlobal('fetch', mockFetchSuccess({ activity_events: newEvents }))
    await act(async () => {
      await result.current.refetch()
    })
    expect(result.current.events[0]!.summary).toBe('Updated')
  })
})
