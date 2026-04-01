import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useWebTTY } from './useWebTTY'

// Track WebSocket instances
let mockWsInstances: MockWebSocket[]

class MockWebSocket {
  static OPEN = 1
  static CONNECTING = 0
  static CLOSED = 2
  readyState = MockWebSocket.CONNECTING
  sent: string[] = []
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  url: string

  constructor(url: string) {
    this.url = url
    mockWsInstances.push(this)
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN
      this.onopen?.()
    }, 0)
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.()
  }
}

describe('useWebTTY', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    mockWsInstances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  it('initializes with connecting status', () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    expect(result.current.status).toBe('connecting')
  })

  it('transitions to connected after WebSocket opens', async () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })
  })

  it('sends auth init and base64 preference on connect', async () => {
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
      expect(mockWsInstances[0]!.sent.length).toBeGreaterThanOrEqual(2)
    })

    const sent = mockWsInstances[0]!.sent
    // First message: auth init with session/pane args
    const initMsg = JSON.parse(sent[0]!)
    expect(initMsg).toHaveProperty('AuthToken', '')
    expect(initMsg.Arguments).toContain('session=s1')
    expect(initMsg.Arguments).toContain('pane=p1')
    // Second message: base64 encoding preference
    expect(sent[1]).toBe('4base64')
  })

  it('provides sendText that base64-encodes', async () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })

    act(() => {
      result.current.sendText('hello')
    })

    const sent = mockWsInstances[0]!.sent
    // Should have init + base64 + input
    expect(sent).toContain(`1${btoa('hello')}`)
  })

  it('calls onOutput for type 1 messages', async () => {
    const onOutput = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onOutput }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    act(() => {
      mockWsInstances[0]!.onmessage?.({ data: `1${btoa('output text')}` })
    })

    expect(onOutput).toHaveBeenCalledWith('output text')
  })

  it('calls onWindowTitle for type 3 messages', async () => {
    const onWindowTitle = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onWindowTitle }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    act(() => {
      mockWsInstances[0]!.onmessage?.({ data: `3${JSON.stringify('My Title')}` })
    })

    expect(onWindowTitle).toHaveBeenCalledWith('My Title')
  })

  it('calls onMetadata for type 7 messages', async () => {
    const onMetadata = vi.fn()
    const metadata = { type: 'tab_rename', data: { summary: 'New Name' } }

    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onMetadata }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    act(() => {
      mockWsInstances[0]!.onmessage?.({ data: `7${btoa(JSON.stringify(metadata))}` })
    })

    expect(onMetadata).toHaveBeenCalledWith(metadata)
  })

  it('calls onTabRename for tab_rename metadata', async () => {
    const onTabRename = vi.fn()
    const metadata = { type: 'tab_rename', data: { summary: 'Renamed Tab' } }

    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onTabRename }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    act(() => {
      mockWsInstances[0]!.onmessage?.({ data: `7${btoa(JSON.stringify(metadata))}` })
    })

    expect(onTabRename).toHaveBeenCalledWith('Renamed Tab')
  })

  it('disconnects on unmount', async () => {
    const { unmount } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    unmount()

    expect(mockWsInstances[0]!.readyState).toBe(MockWebSocket.CLOSED)
  })

  it('returns wsRef for direct access', async () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(result.current.wsRef.current).toBeTruthy()
    })

    expect(result.current.wsRef.current).toBe(mockWsInstances[0])
  })

  it('provides sendResize that sends resize message', async () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })

    act(() => {
      result.current.sendResize(80, 24)
    })

    const sent = mockWsInstances[0]!.sent
    expect(sent).toContain('3{"columns":80,"rows":24}')
  })

  it('provides sendRaw for arbitrary data', async () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })

    act(() => {
      result.current.sendRaw('\x1b[?1004l')
    })

    const sent = mockWsInstances[0]!.sent
    expect(sent).toContain('\x1b[?1004l')
  })

  it('calls onStatusChange callback', async () => {
    const onStatusChange = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onStatusChange }))

    await waitFor(() => {
      expect(onStatusChange).toHaveBeenCalledWith('connected')
    })
  })

  it('does not send when WebSocket is not open', async () => {
    const { result } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    // Before connection opens
    act(() => {
      result.current.sendText('test')
    })

    // After unmount (closed)
    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })

    // The ws should be open now, so this should send
    // But let's test with closed state
    const { result: result2 } = renderHook(() => useWebTTY({ session: 's1', pane: 'p1' }))

    // Close before any send
    await waitFor(() => {
      expect(mockWsInstances.length).toBeGreaterThanOrEqual(2)
    })

    // Force close
    const lastWs = mockWsInstances[mockWsInstances.length - 1]!
    lastWs.readyState = MockWebSocket.CLOSED

    act(() => {
      result2.current.sendText('after-close')
    })

    // Should not have been sent since ws is closed
    expect(lastWs.sent).not.toContain(`1${btoa('after-close')}`)
  })

  it('ignores non-string message data', async () => {
    const onOutput = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onOutput }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    act(() => {
      // ArrayBuffer data should be ignored
      mockWsInstances[0]!.onmessage?.({ data: new ArrayBuffer(8) } as unknown as { data: string })
    })

    expect(onOutput).not.toHaveBeenCalled()
  })

  it('handles malformed base64 gracefully', async () => {
    const onOutput = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onOutput }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    act(() => {
      // Invalid base64 should fall back to raw payload
      mockWsInstances[0]!.onmessage?.({ data: '1not-valid-base64!!!' })
    })

    expect(onOutput).toHaveBeenCalledWith('not-valid-base64!!!')
  })

  it('ignores malformed window title', async () => {
    const onWindowTitle = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onWindowTitle }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    // Should not throw
    act(() => {
      mockWsInstances[0]!.onmessage?.({ data: '3{invalid json' })
    })

    expect(onWindowTitle).not.toHaveBeenCalled()
  })

  it('ignores malformed metadata', async () => {
    const onMetadata = vi.fn()
    renderHook(() => useWebTTY({ session: 's1', pane: 'p1', onMetadata }))

    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })

    // Should not throw
    act(() => {
      mockWsInstances[0]!.onmessage?.({ data: '7{bad}' })
    })

    expect(onMetadata).not.toHaveBeenCalled()
  })

  it('reconnects when reconnect option is enabled', async () => {
    // Use real timers since MockWebSocket uses setTimeout internally
    const originalSetTimeout = globalThis.setTimeout
    let scheduledReconnect: (() => void) | null = null

    // Intercept setTimeout to capture the reconnect callback
    vi.spyOn(globalThis, 'setTimeout').mockImplementation((fn, delay) => {
      if (delay && delay >= 1000) {
        scheduledReconnect = fn as () => void
        return 0 as unknown as ReturnType<typeof setTimeout>
      }
      return originalSetTimeout(fn, delay)
    })

    const { result } = renderHook(() =>
      useWebTTY({
        session: 's1',
        pane: 'p1',
        reconnect: true,
        maxReconnectAttempts: 3,
      }),
    )

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })

    expect(mockWsInstances).toHaveLength(1)

    // Simulate close
    act(() => {
      mockWsInstances[0]!.onclose?.()
    })

    // Should be disconnected
    expect(result.current.status).toBe('disconnected')

    // Trigger the reconnect callback manually
    expect(scheduledReconnect).not.toBeNull()
    await act(async () => {
      scheduledReconnect?.()
    })

    // Should have created a new WebSocket
    await waitFor(() => {
      expect(mockWsInstances.length).toBeGreaterThanOrEqual(2)
    })

    vi.restoreAllMocks()
  })

  it('does not reconnect when reconnect option is disabled', async () => {
    const { result } = renderHook(() =>
      useWebTTY({
        session: 's1',
        pane: 'p1',
        reconnect: false,
      }),
    )

    await waitFor(() => {
      expect(result.current.status).toBe('connected')
    })

    expect(mockWsInstances).toHaveLength(1)

    // Simulate close
    act(() => {
      mockWsInstances[0]!.onclose?.()
    })

    // Verify no reconnect was attempted — instance count stays at 1
    await waitFor(() => {
      expect(mockWsInstances).toHaveLength(1)
    })
  })
})
