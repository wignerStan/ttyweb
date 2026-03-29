import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, type Mock, vi } from 'vitest'
import type { AISession, ApiResponse, ConversationMessage } from './types'
import { useConversation, useConversations } from './useConversations'

// --- Helpers ---

function makeSession(overrides: Partial<AISession> = {}): AISession {
  return {
    id: 'sess-1',
    type: 'claude_code',
    model: 'claude-3-opus',
    title: 'Test Session',
    message_count: 5,
    ...overrides,
  }
}

function makeMessage(overrides: Partial<ConversationMessage> = {}): ConversationMessage {
  return {
    role: 'user',
    content: 'Hello',
    timestamp: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

// Auth header mock
vi.mock('../utils/auth', () => ({
  getAuthHeader: vi.fn(() => 'Basic dGVzdDp0ZXN0'),
}))

// --- Tests ---

describe('useConversations', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch sessions on mount', async () => {
    const sessions = [makeSession(), makeSession({ id: 'sess-2', title: 'Second' })]
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () =>
        Promise.resolve({ success: true, data: sessions } satisfies ApiResponse<AISession[]>),
    })

    const { result } = renderHook(() => useConversations(null))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.sessions).toHaveLength(2)
    expect(result.current.sessions[0].title).toBe('Test Session')
    expect(globalThis.fetch).toHaveBeenCalledWith('/api/ai/sessions', {
      headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
    })
  })

  it('should include project query parameter when projectPath is set', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ success: true, data: [] }),
    })

    renderHook(() => useConversations('/home/user/project'))

    await waitFor(() => {
      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/ai/sessions?project=%2Fhome%2Fuser%2Fproject',
        { headers: { Authorization: 'Basic dGVzdDp0ZXN0' } },
      )
    })
  })

  it('should set error when API returns success=false', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () =>
        Promise.resolve({
          success: false,
          error: 'unauthorized',
          data: [],
        } satisfies ApiResponse<AISession[]>),
    })

    const { result } = renderHook(() => useConversations(null))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('unauthorized')
    expect(result.current.sessions).toEqual([])
  })

  it('should set default error when API returns success=false without error field', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () =>
        Promise.resolve({
          success: false,
          data: [],
        } satisfies ApiResponse<AISession[]>),
    })

    const { result } = renderHook(() => useConversations(null))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Failed to fetch sessions')
  })

  it('should set error on network failure', async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))

    const { result } = renderHook(() => useConversations(null))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Failed to connect to server')
  })

  it('should refetch sessions when refetch is called', async () => {
    const firstBatch = [makeSession({ id: 's1' })]
    const secondBatch = [makeSession({ id: 's1' }), makeSession({ id: 's2' })]
    let callCount = 0

    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++
      const data = callCount === 1 ? firstBatch : secondBatch
      return Promise.resolve({
        json: () => Promise.resolve({ success: true, data }),
      })
    })

    const { result } = renderHook(() => useConversations(null))

    await waitFor(() => {
      expect(result.current.sessions).toHaveLength(1)
    })

    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(result.current.sessions).toHaveLength(2)
    })
  })

  it('should pass auth headers on every request', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ success: true, data: [] }),
    })

    const { result } = renderHook(() => useConversations(null))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const calls = (globalThis.fetch as Mock).mock.calls
    for (const call of calls) {
      expect(call[1]).toHaveProperty('headers')
      expect((call[1] as RequestInit).headers).toEqual({
        Authorization: 'Basic dGVzdDp0ZXN0',
      })
    }
  })
})

describe('useConversation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch conversation messages on mount', async () => {
    const messages = [
      makeMessage({ role: 'user', content: 'Hi' }),
      makeMessage({ role: 'assistant', content: 'Hello!' }),
    ]
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () =>
        Promise.resolve({ success: true, data: messages } satisfies ApiResponse<
          ConversationMessage[]
        >),
    })

    const { result } = renderHook(() => useConversation('sess-1'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.messages).toHaveLength(2)
    expect(result.current.messages[0].role).toBe('user')
    expect(globalThis.fetch).toHaveBeenCalledWith('/api/ai/sessions/sess-1/conversation', {
      headers: { Authorization: 'Basic dGVzdDp0ZXN0' },
    })
  })

  it('should not fetch when sessionId is null', async () => {
    globalThis.fetch = vi.fn()

    renderHook(() => useConversation(null))

    // Wait a tick to let effects run
    await act(async () => {
      await new Promise((r) => setTimeout(r, 0))
    })

    expect(globalThis.fetch).not.toHaveBeenCalled()
  })

  it('should reset state when sessionId changes', async () => {
    const msgs1 = [makeMessage({ content: 'first' })]
    const msgs2 = [makeMessage({ content: 'second' })]
    let callCount = 0

    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++
      const data = callCount <= 1 ? msgs1 : msgs2
      return Promise.resolve({
        json: () => Promise.resolve({ success: true, data }),
      })
    })

    const { result, rerender } = renderHook(({ id }) => useConversation(id), {
      initialProps: { id: 'sess-1' },
    })

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(1)
      expect(result.current.messages[0].content).toBe('first')
    })

    await rerender({ id: 'sess-2' })

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(1)
      expect(result.current.messages[0].content).toBe('second')
    })
  })

  it('should set error when conversation fetch fails', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      json: () =>
        Promise.resolve({
          success: false,
          error: 'not found',
          data: [],
        } satisfies ApiResponse<ConversationMessage[]>),
    })

    const { result } = renderHook(() => useConversation('missing'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('not found')
    expect(result.current.messages).toEqual([])
  })

  it('should set error on network failure for conversation', async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))

    const { result } = renderHook(() => useConversation('sess-1'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Failed to connect to server')
  })

  it('should refetch via refetch method', async () => {
    const first = [makeMessage({ content: 'initial' })]
    const second = [makeMessage({ content: 'refreshed' }), makeMessage({ content: 'new' })]
    let callCount = 0

    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++
      const data = callCount === 1 ? first : second
      return Promise.resolve({
        json: () => Promise.resolve({ success: true, data }),
      })
    })

    const { result } = renderHook(() => useConversation('sess-1'))

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(1)
    })

    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(2)
    })
  })

  it('should call refresh endpoint via refresh method', async () => {
    const refreshed = [makeMessage({ content: 'refreshed data' })]
    let lastEndpoint = ''

    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      lastEndpoint = url
      return Promise.resolve({
        json: () => Promise.resolve({ success: true, data: refreshed }),
      })
    })

    const { result } = renderHook(() => useConversation('sess-1'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    await act(async () => {
      await result.current.refresh()
    })

    expect(lastEndpoint).toContain('/refresh')
    expect(result.current.messages[0].content).toBe('refreshed data')
  })

  it('should set default error when refresh fails without error field', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('conversation')) {
        return Promise.resolve({
          json: () => Promise.resolve({ success: true, data: [makeMessage()] }),
        })
      }
      return Promise.resolve({
        json: () => Promise.resolve({ success: false, data: [] }),
      })
    })

    const { result } = renderHook(() => useConversation('sess-1'))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    await act(async () => {
      await result.current.refresh()
    })

    expect(result.current.error).toBe('Failed to refresh conversation')
  })
})
