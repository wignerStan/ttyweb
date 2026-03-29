import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useReplyInbox } from './useReplyInbox'

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) }))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useReplyInbox', () => {
  it('submitReply calls POST then PUT', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    const { result } = renderHook(() => useReplyInbox())

    await act(async () => {
      await result.current.submitReply('inbox-1', 'study-1', 'approved', 'Looks good')
    })

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock.mock.calls[0]![0]).toContain('/api/butler/approval_replies')
    expect(fetchMock.mock.calls[0]![1]!.method).toBe('POST')
    expect(fetchMock.mock.calls[1]![0]).toContain('/api/butler/inbox_items/inbox-1')
    expect(fetchMock.mock.calls[1]![1]!.method).toBe('PUT')
  })

  it('returns loading and error states', () => {
    const { result } = renderHook(() => useReplyInbox())
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  it('throws on fetch error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
    const { result } = renderHook(() => useReplyInbox())

    let thrown = false
    await act(async () => {
      try {
        await result.current.submitReply('i1', 's1', 'approved', '')
      } catch {
        thrown = true
      }
    })
    expect(thrown).toBe(true)
    expect(result.current.error).toBeTruthy()
  })
})
