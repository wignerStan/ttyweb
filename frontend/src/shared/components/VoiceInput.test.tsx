import { act, render, screen, waitFor } from '@testing-library/react'
import { createRef } from 'react'
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

  function MockWebSocket() {
    currentMockWS = createMockWebSocket()
    return currentMockWS
  }
  vi.stubGlobal('WebSocket', vi.fn(MockWebSocket))
  function MockAudioContext() {
    return mockAudioCtx
  }
  vi.stubGlobal('AudioContext', vi.fn(MockAudioContext))
  window.alert = vi.fn()
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
  let onText: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onText = vi.fn()
  })

  afterEach(() => {
    cleanupMocks()
    vi.restoreAllMocks()
  })

  it('renders mic button in idle state', () => {
    render(<VoiceInput onText={onText} />)

    const btn = screen.getByRole('button')
    expect(btn).toBeInTheDocument()
    expect(btn.className).toContain('idle')
  })

  it('button is disabled when disabled prop is true', () => {
    render(<VoiceInput onText={onText} disabled />)

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
})
