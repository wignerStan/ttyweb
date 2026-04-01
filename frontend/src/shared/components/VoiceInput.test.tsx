import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { createRef } from 'react'
import type { Mock } from 'vitest'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { VoiceInput, type VoiceInputHandle } from './VoiceInput'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

interface MockWS extends Record<string, unknown> {
  _open: () => void
  _message: (data: string) => void
  _close: () => void
}

function createMockWebSocket(): MockWS {
  const handlers: Record<string, EventListener> = {}
  const ws: MockWS = {
    readyState: 0,
    send: vi.fn(),
    close: vi.fn(),
    onopen: null,
    onmessage: null,
    onclose: null,
    onerror: null,
    addEventListener: vi.fn((event: string, handler: EventListener) => {
      handlers[event] = handler
    }),
    removeEventListener: vi.fn(),
    _open() {
      ws.readyState = 1
      if (typeof ws.onopen === 'function') ws.onopen(new Event('open'))
      handlers.open?.(new Event('open'))
    },
    _message(data: string) {
      if (typeof ws.onmessage === 'function') ws.onmessage(new MessageEvent('message', { data }))
      handlers.message?.(new MessageEvent('message', { data }))
    },
    _close() {
      ws.readyState = 3
      if (typeof ws.onclose === 'function') ws.onclose(new CloseEvent('close'))
      handlers.close?.(new CloseEvent('close'))
    },
  }
  return ws
}

let currentMockWS: MockWS | null = null
let originalMediaDevices: Navigator['mediaDevices']

function setupMocks() {
  const mockStream = { getTracks: vi.fn(() => [{ stop: vi.fn() }]) }
  const mockAudioCtx = {
    state: 'running' as string,
    sampleRate: 16000,
    resume: vi.fn().mockResolvedValue(undefined),
    close: vi.fn(),
    createMediaStreamSource: vi.fn(() => ({ connect: vi.fn() })),
    createScriptProcessor: vi.fn(() => ({
      disconnect: vi.fn(),
      connect: vi.fn(),
      onaudioprocess: null,
    })),
    destination: {},
  }

  currentMockWS = null
  originalMediaDevices = navigator.mediaDevices

  Object.defineProperty(navigator, 'mediaDevices', {
    value: { getUserMedia: vi.fn().mockResolvedValue(mockStream) },
    writable: true,
    configurable: true,
  })

  function MockWebSocket(this: Record<string, unknown>) {
    currentMockWS = createMockWebSocket()
    return currentMockWS
  }
  const MockWSConstructor = vi.fn(MockWebSocket) as Mock<
    (this: Record<string, unknown>) => MockWS
  > & {
    OPEN: number
    CLOSED: number
    CONNECTING: number
    CLOSING: number
  }
  MockWSConstructor.OPEN = 1
  MockWSConstructor.CLOSED = 3
  MockWSConstructor.CONNECTING = 0
  MockWSConstructor.CLOSING = 2
  vi.stubGlobal('WebSocket', MockWSConstructor)
  function MockAudioContext(this: Record<string, unknown>) {
    return mockAudioCtx
  }
  vi.stubGlobal('AudioContext', vi.fn(MockAudioContext))
  globalThis.alert = vi.fn()
}

function cleanupMocks() {
  if (originalMediaDevices !== undefined) {
    Object.defineProperty(navigator, 'mediaDevices', {
      value: originalMediaDevices,
      writable: true,
      configurable: true,
    })
  }
}

describe('VoiceInput', () => {
  let onText: Mock<(text: string) => void>

  beforeEach(() => {
    onText = vi.fn<(text: string) => void>()
  })

  afterEach(() => {
    cleanupMocks()
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('renders mic button in idle state', () => {
    render(<VoiceInput onText={onText} />)

    const btn = screen.getByRole('button')
    expect(btn).toBeInTheDocument()
    expect(btn.className).toContain('bg-[var(--zinc-800)]')
  })

  it('button is disabled when disabled prop is true', () => {
    render(<VoiceInput onText={onText} disabled />)

    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('button is disabled during connecting state', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('connecting')
    })

    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('button is disabled during processing state', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('processing')
    })

    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('exposes imperative handle with toggle and status', () => {
    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    expect(ref.current).toBeTruthy()
    expect(ref.current?.status).toBe('idle')
    expect(typeof ref.current?.toggle).toBe('function')
  })

  it('transitions to connecting state when recording starts', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('connecting')
    })
  })

  it('creates WebSocket connection on start', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })
  })

  it('sends start message when WebSocket opens', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    expect(currentMockWS?.send).toHaveBeenCalledWith(expect.stringContaining('"type":"start"'))
  })

  it('transitions to recording when ready message received', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })
  })

  it('sends stop message and transitions to processing when toggled during recording', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('processing')
    })

    expect(currentMockWS?.send).toHaveBeenCalledWith(expect.stringContaining('"type":"stop"'))
  })

  it('does nothing when toggled during connecting state', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('connecting')
    })

    // Toggle again while connecting - should not throw, stays connecting
    await act(async () => {
      ref.current?.toggle()
    })

    expect(ref.current?.status).toBe('connecting')
  })

  it('displays partial transcript text', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'partial', sn: 1, text: 'hello' }))
    })

    expect(screen.getByText('hello')).toBeInTheDocument()
  })

  it('accumulates partial results in order', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    const onPartial = vi.fn()
    render(<VoiceInput ref={ref} onText={onText} onPartial={onPartial} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'partial', sn: 1, text: 'hello ' }))
      currentMockWS?._message(JSON.stringify({ type: 'partial', sn: 2, text: 'world' }))
    })

    expect(screen.getByText('hello world')).toBeInTheDocument()
    expect(onPartial).toHaveBeenCalledWith('hello world')
  })

  it('replaces partial results on pgs=rpl', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    // First partial: sn 0 -> "wrong "
    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'partial', sn: 0, text: 'wrong ' }))
    })

    // Replacement: pgs=rpl, rg=[0,0] removes sn 0, then sn 0 -> "correct"
    await act(async () => {
      currentMockWS?._message(
        JSON.stringify({ type: 'partial', pgs: 'rpl', rg: [0, 0], sn: 0, text: 'correct' }),
      )
    })

    expect(screen.getByText('correct')).toBeInTheDocument()
    expect(screen.queryByText('wrong')).not.toBeInTheDocument()
  })

  it('calls onText with final text when end message received', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'partial', sn: 1, text: 'hello' }))
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'end' }))
    })

    expect(onText).toHaveBeenCalledWith('hello')
  })

  it('does not call onText when final text is empty/whitespace', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'end' }))
    })

    expect(onText).not.toHaveBeenCalled()
  })

  it('resets to idle on error message from server', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'error' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })

    // Partial text should be cleared
    expect(screen.queryByText('hello')).not.toBeInTheDocument()
  })

  it('clears partial text after end message', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'partial', sn: 1, text: 'hello' }))
    })

    expect(screen.getByText('hello')).toBeInTheDocument()

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'end' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })

    expect(screen.queryByText('hello')).not.toBeInTheDocument()
  })

  it('resets to idle on WebSocket error', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('connecting')
    })

    await act(async () => {
      if (typeof currentMockWS?.onerror === 'function') {
        currentMockWS.onerror(new Event('error'))
      }
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })
  })

  it('resets to idle on WebSocket close', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('connecting')
    })

    await act(async () => {
      currentMockWS?._close()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })
  })

  it('shows alert when navigator.mediaDevices is undefined (HTTPS)', async () => {
    // Remove mediaDevices to simulate unsupported browser
    Object.defineProperty(navigator, 'mediaDevices', {
      value: undefined,
      writable: true,
      configurable: true,
    })
    Object.defineProperty(window, 'location', {
      value: { protocol: 'https:', host: 'localhost' },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    expect(globalThis.alert).toHaveBeenCalledWith('当前浏览器不支持麦克风功能。')
  })

  it('shows alert when navigator.mediaDevices is undefined (HTTP)', async () => {
    Object.defineProperty(navigator, 'mediaDevices', {
      value: undefined,
      writable: true,
      configurable: true,
    })
    Object.defineProperty(window, 'location', {
      value: { protocol: 'http:', host: 'localhost' },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    expect(globalThis.alert).toHaveBeenCalledWith(expect.stringContaining('HTTPS'))
  })

  it('shows alert on NotAllowedError from getUserMedia', async () => {
    const permErr = new Error('Permission denied')
    permErr.name = 'NotAllowedError'
    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockRejectedValue(permErr) },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    expect(globalThis.alert).toHaveBeenCalledWith(expect.stringContaining('麦克风权限被拒绝'))
  })

  it('shows alert on NotFoundError from getUserMedia', async () => {
    const permErr = new Error('No device')
    permErr.name = 'NotFoundError'
    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockRejectedValue(permErr) },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    expect(globalThis.alert).toHaveBeenCalledWith(expect.stringContaining('未检测到麦克风'))
  })

  it('shows alert on generic getUserMedia error (HTTP)', async () => {
    const permErr = new Error('Some error')
    permErr.name = 'AbortError'
    Object.defineProperty(window, 'location', {
      value: { protocol: 'http:', host: 'localhost' },
      writable: true,
      configurable: true,
    })
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockRejectedValue(permErr) },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    expect(globalThis.alert).toHaveBeenCalledWith(expect.stringContaining('HTTP'))
  })

  it('returns to idle after getUserMedia error', async () => {
    const permErr = new Error('No mic')
    permErr.name = 'NotAllowedError'
    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockRejectedValue(permErr) },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })
  })

  it('resets to idle after connect timeout', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    // Open the WS to set the connect timeout
    await act(async () => {
      currentMockWS?._open()
    })

    // Advance past the 10 second connect timeout
    await act(async () => {
      vi.advanceTimersByTime(11000)
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })

    vi.useRealTimers()
  })

  it('cleans up media stream tracks on stop', async () => {
    const mockStop = vi.fn()
    const mockStream = { getTracks: vi.fn(() => [{ stop: mockStop }]) }

    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockResolvedValue(mockStream) },
      writable: true,
      configurable: true,
    })

    function MockWebSocket(this: Record<string, unknown>) {
      currentMockWS = createMockWebSocket()
      return currentMockWS
    }
    const wsCtor = vi.fn(MockWebSocket) as Mock<(this: Record<string, unknown>) => MockWS> & {
      OPEN: number
      CLOSED: number
      CONNECTING: number
      CLOSING: number
    }
    wsCtor.OPEN = 1
    wsCtor.CLOSED = 3
    wsCtor.CONNECTING = 0
    wsCtor.CLOSING = 2
    vi.stubGlobal('WebSocket', wsCtor)
    function MockAudioContext(this: Record<string, unknown>) {
      return {
        state: 'running' as string,
        sampleRate: 16000,
        resume: vi.fn().mockResolvedValue(undefined),
        close: vi.fn(),
        createMediaStreamSource: vi.fn(() => ({ connect: vi.fn() })),
        createScriptProcessor: vi.fn(() => ({
          disconnect: vi.fn(),
          connect: vi.fn(),
          onaudioprocess: null,
        })),
        destination: {},
      }
    }
    vi.stubGlobal('AudioContext', vi.fn(MockAudioContext))
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await act(async () => {
      currentMockWS?._close()
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })

    expect(mockStop).toHaveBeenCalled()
  })

  it('triggers button click via onClick handler', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    const btn = screen.getByRole('button')

    // Click should trigger startRecording
    await act(async () => {
      fireEvent.click(btn)
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('connecting')
    })
  })

  it('blurs active textarea on click', async () => {
    setupMocks()

    // Create a focused textarea
    const textarea = document.createElement('textarea')
    document.body.appendChild(textarea)
    textarea.focus()
    expect(document.activeElement).toBe(textarea)

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    const btn = screen.getByRole('button')

    await act(async () => {
      fireEvent.click(btn)
    })

    // After clicking voice button, textarea should be blurred
    expect(document.activeElement).not.toBe(textarea)
  })

  it('handles unknown message type gracefully', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })

    // Send unknown message type - should not throw
    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'unknown_type' }))
    })

    // Should still be recording
    expect(ref.current?.status).toBe('recording')
  })

  it('handles catch block in startRecording', async () => {
    // Mock getUserMedia to throw a non-Error
    Object.defineProperty(navigator, 'mediaDevices', {
      value: {
        getUserMedia: vi.fn().mockImplementation(() => {
          throw 'unexpected string error'
        }),
      },
      writable: true,
      configurable: true,
    })
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    // Should not have alerted since it's not an Error instance
    expect(globalThis.alert).not.toHaveBeenCalled()
    await waitFor(() => {
      expect(ref.current?.status).toBe('idle')
    })
  })

  it('cancels connect timeout when ready message received', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    // Receive ready before timeout
    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })

    // Advance past timeout - should NOT reset since ready was received
    await act(async () => {
      vi.advanceTimersByTime(11000)
    })

    // Should still be recording (timeout was cleared)
    expect(ref.current?.status).toBe('recording')
    vi.useRealTimers()
  })

  it('uses webkitAudioContext fallback when AudioContext is not available', async () => {
    const mockStream = { getTracks: vi.fn(() => [{ stop: vi.fn() }]) }
    const mockAudioCtx = {
      state: 'running' as string,
      sampleRate: 16000,
      resume: vi.fn().mockResolvedValue(undefined),
      close: vi.fn(),
      createMediaStreamSource: vi.fn(() => ({ connect: vi.fn() })),
      createScriptProcessor: vi.fn(() => ({
        disconnect: vi.fn(),
        connect: vi.fn(),
        onaudioprocess: null,
      })),
      destination: {},
    }

    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockResolvedValue(mockStream) },
      writable: true,
      configurable: true,
    })

    // Only provide webkitAudioContext
    vi.stubGlobal('AudioContext', undefined)
    function MockWebKitAudioContext(this: Record<string, unknown>) {
      return mockAudioCtx
    }
    ;(globalThis as unknown as Record<string, unknown>).webkitAudioContext =
      vi.fn(MockWebKitAudioContext)

    function MockWebSocket(this: Record<string, unknown>) {
      currentMockWS = createMockWebSocket()
      return currentMockWS
    }
    const wsCtor2 = vi.fn(MockWebSocket) as Mock<(this: Record<string, unknown>) => MockWS> & {
      OPEN: number
      CLOSED: number
      CONNECTING: number
      CLOSING: number
    }
    wsCtor2.OPEN = 1
    wsCtor2.CLOSED = 3
    wsCtor2.CONNECTING = 0
    wsCtor2.CLOSING = 2
    vi.stubGlobal('WebSocket', wsCtor2)
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    // Should have reached recording state using webkitAudioContext
    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })
  })

  it('resumes suspended audio context', async () => {
    const mockResume = vi.fn().mockResolvedValue(undefined)
    const mockStream = { getTracks: vi.fn(() => [{ stop: vi.fn() }]) }
    const mockAudioCtx = {
      state: 'suspended' as string,
      sampleRate: 16000,
      resume: mockResume,
      close: vi.fn(),
      createMediaStreamSource: vi.fn(() => ({ connect: vi.fn() })),
      createScriptProcessor: vi.fn(() => ({
        disconnect: vi.fn(),
        connect: vi.fn(),
        onaudioprocess: null,
      })),
      destination: {},
    }

    originalMediaDevices = navigator.mediaDevices
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia: vi.fn().mockResolvedValue(mockStream) },
      writable: true,
      configurable: true,
    })

    function MockWebSocket(this: Record<string, unknown>) {
      currentMockWS = createMockWebSocket()
      return currentMockWS
    }
    const wsCtor3 = vi.fn(MockWebSocket) as Mock<(this: Record<string, unknown>) => MockWS> & {
      OPEN: number
      CLOSED: number
      CONNECTING: number
      CLOSING: number
    }
    wsCtor3.OPEN = 1
    wsCtor3.CLOSED = 3
    wsCtor3.CONNECTING = 0
    wsCtor3.CLOSING = 2
    vi.stubGlobal('WebSocket', wsCtor3)
    function MockAudioContext(this: Record<string, unknown>) {
      return mockAudioCtx
    }
    vi.stubGlobal('AudioContext', vi.fn(MockAudioContext))
    globalThis.alert = vi.fn()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    await act(async () => {
      currentMockWS?._open()
    })

    await act(async () => {
      currentMockWS?._message(JSON.stringify({ type: 'ready' }))
    })

    await waitFor(() => {
      expect(ref.current?.status).toBe('recording')
    })

    expect(mockResume).toHaveBeenCalled()
  })

  it('connects to correct WebSocket URL with auth header', async () => {
    setupMocks()

    const ref = createRef<VoiceInputHandle | null>()
    render(<VoiceInput ref={ref} onText={onText} />)

    await act(async () => {
      ref.current?.toggle()
    })

    await waitFor(() => {
      expect(currentMockWS).toBeTruthy()
    })

    // Verify WebSocket was called with URL containing /ws/speech and auth token
    const wsConstructor = globalThis.WebSocket as unknown as Mock
    expect(wsConstructor).toHaveBeenCalledWith(expect.stringContaining('/ws/speech'))
    expect(wsConstructor).toHaveBeenCalledWith(expect.stringContaining('Bearer%20test-token'))
  })
})
