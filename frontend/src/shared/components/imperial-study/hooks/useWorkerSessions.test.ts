import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mockFetchHttpError, mockFetchSuccess } from '../../../../test-helpers'
import { useWorkerSessions } from './useWorkerSessions'

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

const workers = [
  {
    id: 'w1',
    study_id: 's1',
    session_id: 'agent-1',
    pane_target: 'butler/quant:%1',
    port: 9001,
    state: 'idle',
    run_id: '',
    project: 'proj',
    workdir: '/tmp',
    last_seen_at: null,
    created_at: null,
    updated_at: null,
  },
]

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetchSuccess({ worker_sessions: workers }))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useWorkerSessions', () => {
  it('fetches workers on mount from correct URL', async () => {
    const { result } = renderHook(() => useWorkerSessions())
    expect(fetch).toHaveBeenCalledWith('/api/butler/worker_sessions', expect.any(Object))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.workers).toEqual(workers)
  })

  it('passes study_id filter', async () => {
    renderHook(() => useWorkerSessions('s1'))
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('study_id=s1'), expect.any(Object))
  })

  it('handles error', async () => {
    vi.stubGlobal('fetch', mockFetchHttpError(500))
    const { result } = renderHook(() => useWorkerSessions())
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toBeTruthy()
    expect(result.current.workers).toEqual([])
  })

  it('refetch reloads data', async () => {
    const { result } = renderHook(() => useWorkerSessions())
    await waitFor(() => expect(result.current.loading).toBe(false))
    const updatedWorkers = [{ ...workers[0], state: 'busy' as const }]
    vi.stubGlobal('fetch', mockFetchSuccess({ worker_sessions: updatedWorkers }))
    await act(async () => {
      await result.current.refetch()
    })
    expect(result.current.workers[0].state).toBe('busy')
  })
})
