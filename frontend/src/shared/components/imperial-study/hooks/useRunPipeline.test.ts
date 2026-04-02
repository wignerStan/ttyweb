import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { RoutingInfo } from '../types'
import { useRunPipeline } from './useRunPipeline'

vi.mock('../../../../utils/auth', () => ({
  getAuthHeader: () => '',
  getAuthHeaders: () => ({}),
}))

const dashboardRuns = [
  {
    id: 'r1',
    task_id: 't1',
    state: 'running',
    assistant: 'opencode',
    intent: 'do thing',
    queued_at: null,
  },
]

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ data: { runs: dashboardRuns } }),
    }),
  )
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useRunPipeline', () => {
  it('fetches pipeline runs on mount', async () => {
    const { result } = renderHook(() => useRunPipeline())
    expect(fetch).toHaveBeenCalledWith('/api/butler/dashboard/runs?limit=5', expect.any(Object))
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.runs[0]!.run_id).toBe('r1')
    expect(result.current.runs[0]!.stage).toBe('processing')
  })

  it('computes activeRun from non-terminal runs', async () => {
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.activeRun).not.toBeNull()
    expect(result.current.activeRun?.run_id).toBe('r1')
  })

  it('dispatch adds a new run', async () => {
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))

    const routing: RoutingInfo = { strategy: 'assistant', executor: 'opencode', delegated: false }
    act(() => {
      result.current.dispatch('new intent', routing, 'r2', 't2')
    })
    expect(result.current.runs[0]!.run_id).toBe('r2')
    expect(result.current.runs).toHaveLength(2)
  })

  it('dismiss removes a run', async () => {
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    act(() => {
      result.current.dismiss('r1')
    })
    expect(result.current.runs).toHaveLength(0)
  })

  it('activeRun is null when all runs are terminal', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({ data: { runs: [{ ...dashboardRuns[0], state: 'succeeded' }] } }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.activeRun).toBeNull()
  })

  // --- mapState branches via dashboard fetch ---

  it('maps "succeeded" state to return/success', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            data: { runs: [{ ...dashboardRuns[0]!, state: 'succeeded' }] },
          }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.runs[0]!.stage).toBe('return')
    expect(result.current.runs[0]!.status).toBe('success')
  })

  it('maps "failed" state to return/failed', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            data: { runs: [{ ...dashboardRuns[0]!, state: 'failed' }] },
          }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.runs[0]!.stage).toBe('return')
    expect(result.current.runs[0]!.status).toBe('failed')
  })

  it('maps "cancelled" state to return/failed', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            data: { runs: [{ ...dashboardRuns[0]!, state: 'cancelled' }] },
          }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.runs[0]!.stage).toBe('return')
    expect(result.current.runs[0]!.status).toBe('failed')
  })

  it('maps unknown state to outflow/pending', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            data: { runs: [{ ...dashboardRuns[0]!, state: 'queued' }] },
          }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.runs[0]!.stage).toBe('outflow')
    expect(result.current.runs[0]!.status).toBe('pending')
  })

  // --- dashboardToPipelineRun null fields ---

  it('handles null assistant and intent in dashboard run', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            data: {
              runs: [{ ...dashboardRuns[0]!, assistant: null, intent: null, queued_at: null }],
            },
          }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => expect(result.current.runs).toHaveLength(1))
    expect(result.current.runs[0]!.intent).toBe('')
    expect(result.current.runs[0]!.routing.strategy).toBe('unknown')
    expect(result.current.runs[0]!.routing.executor).toBe('')
  })

  // --- initial fetch error paths ---

  it('handles non-ok response from dashboard fetch', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => {
      expect(result.current.runs).toHaveLength(0)
    }, { timeout: 1000 })
  })

  it('handles network error during dashboard fetch', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('Network error')))
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => {
      expect(result.current.runs).toHaveLength(0)
    }, { timeout: 1000 })
  })

  it('handles empty runs array from dashboard', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ data: { runs: [] } }),
      }),
    )
    const { result } = renderHook(() => useRunPipeline())
    await waitFor(() => {
      expect(result.current.runs).toHaveLength(0)
    }, { timeout: 1000 })
  })
})
