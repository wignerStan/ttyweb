import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AiConversation } from '../types'

// Mock auth utility at module level
vi.mock('../utils/auth', () => ({
  getAuthHeader: vi.fn(() => null),
}))

import { getAuthHeader } from '../utils/auth'
import { useAIConversations } from './useAIConversations'

const mockGetAuthHeader = vi.mocked(getAuthHeader)

// Track EventSource instances
let mockEventSourceInstances: Array<{
  url: string
  close: ReturnType<typeof vi.fn>
  onmessage: ((event: MessageEvent) => void) | null
  onerror: (() => void) | null
}>

class MockEventSource {
  url: string
  close = vi.fn()
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null

  constructor(url: string) {
    this.url = url
    mockEventSourceInstances.push(this)
  }
}

describe('useAIConversations', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    mockEventSourceInstances = []
    vi.stubGlobal('EventSource', MockEventSource)
  })

  it('returns empty conversations and not loading when paneKey is null', () => {
    const { result } = renderHook(() => useAIConversations(null))

    expect(result.current.conversations).toEqual([])
    expect(result.current.loading).toBe(false)
  })

  it('fetches conversations when paneKey is provided', async () => {
    const mockConversations: AiConversation[] = [
      {
        conversation_id: 'conv1',
        pane_key: 'pane1',
        user_message: 'hello',
        assistant_message: 'hi there',
        conv_status: 'completed',
        started_at: 1000,
        completed_at: 2000,
      },
    ]

    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: mockConversations }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAIConversations('pane1'))

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.conversations).toEqual(mockConversations)
    expect(mockFetch).toHaveBeenCalledWith('/api/tasks/events/pane1', { headers: undefined })
  })

  it('creates EventSource with correct URL when paneKey is provided', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockEventSourceInstances).toHaveLength(1)
    })

    expect(mockEventSourceInstances[0]!.url).toContain('/api/tasks/events/stream/pane1')
  })

  it('includes auth token in EventSource URL', async () => {
    mockGetAuthHeader.mockReturnValue('Basic token123')

    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockEventSourceInstances).toHaveLength(1)
    })

    expect(mockEventSourceInstances[0]!.url).toContain('auth=Basic%20token123')
  })

  it('refetches conversations on task events from SSE', async () => {
    const mockConversations: AiConversation[] = [
      {
        conversation_id: 'conv2',
        pane_key: 'pane1',
        user_message: 'new message',
        assistant_message: 'new response',
        conv_status: 'completed',
        started_at: 3000,
        completed_at: 4000,
      },
    ]

    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: mockConversations }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    // Wait for initial fetch
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    // Simulate SSE message
    act(() => {
      const es = mockEventSourceInstances[0]
      if (es?.onmessage) {
        es.onmessage(
          new MessageEvent('message', {
            data: JSON.stringify({ type: 'task_completed' }),
          }),
        )
      }
    })

    // Should have refetched
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(2)
    })
  })

  it('refetches on various task event types', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    const taskEvents = ['task_started', 'task_completed', 'task_failed', 'task_waiting']

    for (const eventType of taskEvents) {
      act(() => {
        const es = mockEventSourceInstances[0]
        if (es?.onmessage) {
          es.onmessage(
            new MessageEvent('message', {
              data: JSON.stringify({ type: eventType }),
            }),
          )
        }
      })

      await waitFor(() => {
        expect(mockFetch.mock.calls.length).toBeGreaterThan(1)
      })
    }
  })

  it('ignores non-task SSE events', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    // Send a non-task event
    act(() => {
      const es = mockEventSourceInstances[0]
      if (es?.onmessage) {
        es.onmessage(
          new MessageEvent('message', {
            data: JSON.stringify({ type: 'other_event' }),
          }),
        )
      }
    })

    // Should NOT have refetched
    expect(mockFetch).toHaveBeenCalledTimes(1)
  })

  it('handles malformed SSE data gracefully', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    // Send malformed JSON
    act(() => {
      const es = mockEventSourceInstances[0]
      if (es?.onmessage) {
        es.onmessage(
          new MessageEvent('message', {
            data: 'not valid json',
          }),
        )
      }
    })

    // Should not crash, and should not refetch
    expect(mockFetch).toHaveBeenCalledTimes(1)
  })

  it('handles SSE onerror gracefully', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockEventSourceInstances).toHaveLength(1)
    })

    // Trigger onerror
    act(() => {
      const es = mockEventSourceInstances[0]
      if (es?.onerror) {
        es.onerror()
      }
    })

    // Should not crash
    expect(true).toBe(true)
  })

  it('closes EventSource on unmount', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { unmount } = renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockEventSourceInstances).toHaveLength(1)
    })

    unmount()

    expect(mockEventSourceInstances[0]!.close).toHaveBeenCalled()
  })

  it('clears conversations when paneKey changes to null', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result, rerender } = renderHook(
      ({ paneKey }: { paneKey: string | null }) => useAIConversations(paneKey),
      {
        initialProps: { paneKey: 'pane1' as string | null },
      },
    )

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    // Change to null
    rerender({ paneKey: null })

    expect(result.current.conversations).toEqual([])
    expect(result.current.loading).toBe(false)
  })

  it('sends auth header when fetching conversations', async () => {
    mockGetAuthHeader.mockReturnValue('Basic mytoken')

    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/tasks/events/pane1', {
        headers: { Authorization: 'Basic mytoken' },
      })
    })
  })

  it('handles fetch error gracefully', async () => {
    const mockFetch = vi.fn().mockRejectedValue(new Error('Network error'))
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAIConversations('pane1'))

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.conversations).toEqual([])
  })

  it('exposes refetch function', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAIConversations('pane1'))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    // Call refetch
    await act(async () => {
      await result.current.refetch()
    })

    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it('does not fetch when refetch is called with null paneKey', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    const { result } = renderHook(() => useAIConversations(null))

    await act(async () => {
      await result.current.refetch()
    })

    expect(mockFetch).not.toHaveBeenCalled()
  })

  it('encodes paneKey in URL', async () => {
    mockGetAuthHeader.mockReturnValue(null)

    const mockFetch = vi.fn().mockResolvedValue({
      json: () => Promise.resolve({ conversations: [] }),
    })
    vi.stubGlobal('fetch', mockFetch)

    renderHook(() => useAIConversations('pane with spaces'))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/tasks/events/pane%20with%20spaces', {
        headers: undefined,
      })
    })
  })
})
