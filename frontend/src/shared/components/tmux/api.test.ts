import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  assignSessionGroup,
  createGroup,
  fetchPaneStatuses,
  fetchProfileOrder,
  fetchTaskPaneStatuses,
  rebuildSession,
  renameWindow,
  saveOrder,
} from './api'

// ── Mocks ──────────────────────────────────────────────────────────────────

vi.mock('../../../utils/api', () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiPut: vi.fn(),
}))

vi.mock('../../../utils/auth', () => ({
  getAuthHeaders: vi.fn(() => ({ Authorization: 'Basic dGVzdDp0ZXN0' })),
}))

import { apiGet, apiPost, apiPut } from '../../../utils/api'

// ── renameWindow ───────────────────────────────────────────────────────────

describe('renameWindow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns false when apiPut resolves null (void success)', async () => {
    // KNOWN DEFECT: apiPut<void> returns null for json.data on successful void endpoints.
    // renameWindow checks res !== null, so it returns false even on success.
    vi.mocked(apiPut).mockResolvedValueOnce(null)

    const result = await renameWindow('main', 2, 'editor')

    expect(apiPut).toHaveBeenCalledWith('/api/tmux/windows/main/2/rename', { name: 'editor' })
    expect(result).toBe(false)
  })

  it('returns true when apiPut returns a non-null value', async () => {
    vi.mocked(apiPut).mockResolvedValueOnce({} as unknown as undefined)

    const result = await renameWindow('main', 2, 'editor')

    expect(result).toBe(true)
  })

  it('encodes special characters in session name', async () => {
    vi.mocked(apiPut).mockResolvedValueOnce({} as Record<string, unknown>)

    await renameWindow('my session', 0, 'new name')

    expect(apiPut).toHaveBeenCalledWith('/api/tmux/windows/my%20session/0/rename', {
      name: 'new name',
    })
  })
})

// ── rebuildSession ─────────────────────────────────────────────────────────

describe('rebuildSession', () => {
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    vi.clearAllMocks()
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('returns ok:true on successful response', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
    } as Response)

    const result = await rebuildSession('main')

    expect(result).toEqual({ ok: true })
    expect(fetch).toHaveBeenCalledWith(
      '/api/tmux/sessions/main/rebuild',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('returns ok:false with server message on error response', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      json: () => Promise.resolve({ message: 'Session not found' }),
    } as unknown as Response)

    const result = await rebuildSession('main')

    expect(result).toEqual({ ok: false, message: 'Session not found' })
  })

  it('returns ok:false with "Unknown error" when server message is missing', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      json: () => Promise.resolve({}),
    } as unknown as Response)

    const result = await rebuildSession('main')

    expect(result).toEqual({ ok: false, message: 'Unknown error' })
  })

  it('returns ok:false when JSON parsing fails', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      json: () => Promise.reject(new SyntaxError('bad json')),
    } as unknown as Response)

    const result = await rebuildSession('main')

    // .catch(() => ({})) swallows the error, then data.message is undefined
    expect(result).toEqual({ ok: false, message: 'Unknown error' })
  })

  it('returns ok:false with Error message on network error', async () => {
    globalThis.fetch = vi.fn().mockRejectedValueOnce(new Error('Connection refused'))

    const result = await rebuildSession('main')

    expect(result).toEqual({ ok: false, message: 'Connection refused' })
  })

  it('returns ok:false with "Network error" on non-Error network error', async () => {
    globalThis.fetch = vi.fn().mockRejectedValueOnce('string error')

    const result = await rebuildSession('main')

    expect(result).toEqual({ ok: false, message: 'Network error' })
  })

  it('encodes special characters in session name', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({ ok: true } as Response)

    await rebuildSession('my session')

    expect(fetch).toHaveBeenCalledWith(
      '/api/tmux/sessions/my%20session/rebuild',
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('includes auth and content-type headers', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({ ok: true } as Response)

    await rebuildSession('main')

    expect(fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Basic dGVzdDp0ZXN0',
        },
      }),
    )
  })
})

// ── saveOrder ──────────────────────────────────────────────────────────────

describe('saveOrder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('calls apiPut with correct URL and body', async () => {
    vi.mocked(apiPut).mockResolvedValueOnce(null)

    const orderData = {
      groups: [{ id: 1, sort_order: 0 }],
      sessions: [
        { session_name: 's1', group_id: 1, sort_order: 0 },
        { session_name: 's2', group_id: null, sort_order: 1 },
      ],
    }

    await saveOrder(5, orderData)

    expect(apiPut).toHaveBeenCalledWith('/api/profiles/5/order', orderData)
  })
})

// ── fetchProfileOrder ──────────────────────────────────────────────────────

describe('fetchProfileOrder', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns profile order on success', async () => {
    const mockData = {
      groups: [
        {
          id: 1,
          sort_order: 0,
          sessions: [{ session_name: 's1', sort_order: 0 }],
        },
      ],
      ungrouped: [{ session_name: 's2', sort_order: 1 }],
    }
    vi.mocked(apiGet).mockResolvedValueOnce(mockData)

    const result = await fetchProfileOrder(3)

    expect(apiGet).toHaveBeenCalledWith('/api/profiles/3/order')
    expect(result).toEqual(mockData)
  })

  it('returns null when apiGet returns null', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce(null)

    const result = await fetchProfileOrder(3)

    expect(result).toBeNull()
  })
})

// ── fetchPaneStatuses ──────────────────────────────────────────────────────

describe('fetchPaneStatuses', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns empty array when paneKeys is empty', async () => {
    const result = await fetchPaneStatuses('profile1', [])

    expect(result).toEqual([])
    expect(apiGet).not.toHaveBeenCalled()
  })

  it('returns pane statuses on success', async () => {
    const mockPanes = [
      { paneKey: 'pane1', status: 'idle' as const, mtime: 1000 },
      { paneKey: 'pane2', status: 'in_progress' as const, mtime: 2000 },
    ]
    vi.mocked(apiGet).mockResolvedValueOnce({ panes: mockPanes })

    const result = await fetchPaneStatuses('profile1', ['pane1', 'pane2'])

    expect(apiGet).toHaveBeenCalledWith(
      '/api/panes/status?profile_key=profile1&paneKeys=pane1,pane2',
    )
    expect(result).toEqual(mockPanes)
  })

  it('returns empty array when apiGet returns null', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce(null)

    const result = await fetchPaneStatuses('profile1', ['pane1'])

    expect(result).toEqual([])
  })

  it('encodes special characters in keys', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({ panes: [] })

    await fetchPaneStatuses('my profile', ['pane:1', 'pane 2'])

    expect(apiGet).toHaveBeenCalledWith(
      '/api/panes/status?profile_key=my%20profile&paneKeys=pane%3A1,pane%202',
    )
  })

  it('returns empty array when apiGet returns data without panes', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({})

    const result = await fetchPaneStatuses('profile1', ['pane1'])

    expect(result).toEqual([])
  })
})

// ── fetchTaskPaneStatuses ──────────────────────────────────────────────────

describe('fetchTaskPaneStatuses', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns empty object when apiGet returns null', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce(null)

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({})
  })

  it('maps tasks to pane status record', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [
        { pane_key: 'pane1', task_status: 'in_progress' },
        { pane_key: 'pane2', task_status: 'done' },
      ],
    })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({
      pane1: 'in_progress',
      pane2: 'done',
    })
  })

  it('normalizes "completed" to "done"', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [{ pane_key: 'pane1', task_status: 'completed' }],
    })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({ pane1: 'done' })
  })

  it('uses higher-priority status when multiple tasks share a pane_key', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [
        { pane_key: 'pane1', task_status: 'done' },
        { pane_key: 'pane1', task_status: 'in_progress' },
        { pane_key: 'pane1', task_status: 'waiting' },
      ],
    })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({ pane1: 'in_progress' })
  })

  it('keeps existing higher-priority status over new lower-priority one', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [
        { pane_key: 'pane1', task_status: 'failed' },
        { pane_key: 'pane1', task_status: 'done' },
      ],
    })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({ pane1: 'failed' })
  })

  it('skips tasks with empty pane_key', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [
        { pane_key: '', task_status: 'in_progress' },
        { pane_key: 'pane1', task_status: 'done' },
      ],
    })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({ pane1: 'done' })
  })

  it('handles unknown status with priority 0', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [{ pane_key: 'pane1', task_status: 'unknown_status' }],
    })

    const result = await fetchTaskPaneStatuses()

    // unknown_status has priority 0, which is > priority of '' (also 0)
    // but 0 is NOT > 0, so it won't be added
    expect(result).toEqual({})
  })

  it('returns empty object for empty tasks array', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({ tasks: [] })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({})
  })

  it('handles null tasks array by treating it as empty', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({ tasks: null })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({})
  })

  it('handles tasks with null task_status', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [{ pane_key: 'pane1', task_status: null as unknown as string }],
    })

    const result = await fetchTaskPaneStatuses()

    // null ?? '' = '', priority[''] = undefined ?? 0 = 0
    // 0 > 0 is false, so nothing added
    expect(result).toEqual({})
  })

  it('replaces lower-priority status with higher one', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [
        { pane_key: 'pane1', task_status: 'waiting' },
        { pane_key: 'pane1', task_status: 'failed' },
      ],
    })

    const result = await fetchTaskPaneStatuses()

    expect(result).toEqual({ pane1: 'failed' })
  })

  it('does not replace with equal-priority status', async () => {
    vi.mocked(apiGet).mockResolvedValueOnce({
      tasks: [
        { pane_key: 'pane1', task_status: 'done' },
        { pane_key: 'pane1', task_status: 'completed' },
      ],
    })

    const result = await fetchTaskPaneStatuses()

    // done has priority 1, completed has priority 1
    // 1 > 1 is false, so first one stays
    expect(result).toEqual({ pane1: 'done' })
  })
})

// ── assignSessionGroup ─────────────────────────────────────────────────────

describe('assignSessionGroup', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('calls apiPut with group id', async () => {
    vi.mocked(apiPut).mockResolvedValueOnce(null)

    await assignSessionGroup('main', 'profile1', 5)

    expect(apiPut).toHaveBeenCalledWith('/api/sessions/main/group', {
      profile_key: 'profile1',
      group_id: 5,
    })
  })

  it('calls apiPut with null group id', async () => {
    vi.mocked(apiPut).mockResolvedValueOnce(null)

    await assignSessionGroup('main', 'profile1', null)

    expect(apiPut).toHaveBeenCalledWith('/api/sessions/main/group', {
      profile_key: 'profile1',
      group_id: null,
    })
  })

  it('encodes special characters in session name', async () => {
    vi.mocked(apiPut).mockResolvedValueOnce(null)

    await assignSessionGroup('my session', 'profile1', 1)

    expect(apiPut).toHaveBeenCalledWith('/api/sessions/my%20session/group', {
      profile_key: 'profile1',
      group_id: 1,
    })
  })
})

// ── createGroup ────────────────────────────────────────────────────────────

describe('createGroup', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns result with id on success', async () => {
    vi.mocked(apiPost).mockResolvedValueOnce({ id: 42 })

    const result = await createGroup('profile1', 'my group')

    expect(apiPost).toHaveBeenCalledWith('/api/groups', {
      profile_key: 'profile1',
      group_name: 'my group',
    })
    expect(result).toEqual({ id: 42 })
  })

  it('returns empty object when apiPost returns null', async () => {
    vi.mocked(apiPost).mockResolvedValueOnce(null)

    const result = await createGroup('profile1', 'my group')

    expect(result).toEqual({})
  })
})
