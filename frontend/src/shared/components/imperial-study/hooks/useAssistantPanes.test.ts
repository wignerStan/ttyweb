import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useAssistantPanes } from './useAssistantPanes'

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

function createMockStream(chunks: string[]): { body: ReadableStream<Uint8Array> } {
  const encoder = new TextEncoder()
  let i = 0
  const stream = new ReadableStream({
    pull(controller) {
      if (i < chunks.length) {
        controller.enqueue(encoder.encode(chunks[i]))
        i++
      } else {
        controller.close()
      }
    },
  })
  return { body: stream }
}

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn())
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useAssistantPanes', () => {
  it('sendMessage creates user and assistant messages', async () => {
    const { body } = createMockStream([
      'event: chunk\ndata: {"content":"hello"}\n\n',
      'event: done\ndata: {"content":"hello world"}\n\n',
    ])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, body }))

    const { result } = renderHook(() => useAssistantPanes())

    await act(async () => {
      result.current.sendMessage('hi', 'chat')
    })

    expect(result.current.messages).toHaveLength(2)
    expect(result.current.messages[0].role).toBe('user')
    expect(result.current.messages[1].role).toBe('assistant')

    await waitFor(() => expect(result.current.streaming).toBe(false))
    expect(result.current.messages[1].content).toBe('hello world')
  })

  it('clearMessages resets state', async () => {
    const { body } = createMockStream(['event: done\ndata: {}\n\n'])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, body }))

    const { result } = renderHook(() => useAssistantPanes())
    await act(async () => {
      result.current.sendMessage('hi', 'chat')
    })
    expect(result.current.messages.length).toBeGreaterThan(0)

    act(() => {
      result.current.clearMessages()
    })
    expect(result.current.messages).toEqual([])
    expect(result.current.streaming).toBe(false)
  })

  it('handles fetch error response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500, body: null }))
    const { result } = renderHook(() => useAssistantPanes())

    await act(async () => {
      result.current.sendMessage('hi', 'chat')
    })
    await waitFor(() => expect(result.current.streaming).toBe(false))
    expect(result.current.messages[1].error).toBe('HTTP 500')
  })

  it('handles reasoning events', async () => {
    const { body } = createMockStream([
      'event: reasoning\ndata: {"content":"think"}\n\n',
      'event: chunk\ndata: {"content":"answer"}\n\n',
      'event: done\ndata: {}\n\n',
    ])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, body }))

    const { result } = renderHook(() => useAssistantPanes())
    await act(async () => {
      result.current.sendMessage('hi', 'chat')
    })
    await waitFor(() => expect(result.current.streaming).toBe(false))
    expect(result.current.messages[1].reasoning).toBe('think')
    expect(result.current.messages[1].content).toBe('answer')
  })
})
