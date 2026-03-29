import { renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useRunDetail } from './useRunDetail'

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

const mockRun = {
  id: 'run-123',
  task_id: 'task-1',
  state: 'running',
  trigger: null,
  attempt: 1,
  input_data: { intent: 'do thing' },
  result: null,
  error: null,
  queued_at: null,
  started_at: null,
  ended_at: null,
  estimated_at: null,
}
const mockEvents = [
  {
    id: 1,
    run_id: 'run-123',
    event_type: 'task_started',
    payload: null,
    created_at: '2026-01-01T00:00:00Z',
  },
]

function mockFetchRun(runData: unknown, eventsData: unknown) {
  return vi.fn((url: string) => {
    if (url.includes('/events')) {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({ data: eventsData }) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve({ data: runData }) })
  })
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useRunDetail', () => {
  it('fetches run detail and events in parallel', async () => {
    vi.stubGlobal('fetch', mockFetchRun(mockRun, { events: mockEvents }))
    const { result } = renderHook(() => useRunDetail('run-123'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.run).toEqual(mockRun)
    expect(result.current.events).toEqual(mockEvents)
  })

  it('shows loading state initially', () => {
    vi.stubGlobal('fetch', () => new Promise(() => {}))
    const { result } = renderHook(() => useRunDetail('run-123'))
    expect(result.current.loading).toBe(true)
  })

  it('sets error on fetch failure', async () => {
    vi.stubGlobal('fetch', () => Promise.resolve({ ok: false, status: 500 }))
    const { result } = renderHook(() => useRunDetail('run-123'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toBeTruthy()
  })

  it('clears state when runId is null', async () => {
    vi.stubGlobal('fetch', mockFetchRun(mockRun, { events: [] }))
    const { result, rerender } = renderHook(({ id }) => useRunDetail(id), {
      initialProps: { id: 'run-123' },
    })
    await waitFor(() => expect(result.current.run).toBeTruthy())
    rerender({ id: null })
    expect(result.current.run).toBeNull()
    expect(result.current.events).toEqual([])
  })

  it('returns null for null runId', () => {
    const { result } = renderHook(() => useRunDetail(null))
    expect(result.current.run).toBeNull()
    expect(result.current.loading).toBe(false)
  })
})
