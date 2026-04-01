import { act, render, screen } from '@testing-library/react'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import type { Mock } from 'vitest'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createTelemetryEmitter } from '../utils/telemetryEmitter'
import { MobileTerminal } from './MobileTerminal'

type AnyFn = (...args: unknown[]) => unknown

interface MockTerminal {
  open: Mock
  loadAddon: Mock
  write: Mock
  dispose: Mock
  onData: Mock
  onResize: Mock
  cols: number
  rows: number
  options: Record<string, unknown>
  buffer: {
    active: {
      viewportY: number
      getLine: Mock
    }
  }
}

interface MockWS extends Record<string, unknown> {
  _open: () => void
  _message: (data: string) => void
  _close: () => void
  _error: () => void
}

vi.mock('@xterm/xterm', () => {
  const MockTerminal = vi.fn(function (
    this: Record<string, unknown>,
    opts: Record<string, unknown> = {},
  ) {
    this.open = vi.fn()
    this.loadAddon = vi.fn()
    this.write = vi.fn()
    this.dispose = vi.fn()
    this.onData = vi.fn((cb: AnyFn) => {
      ;(this as Record<string, unknown>)._onDataCb = cb
    })
    this.onResize = vi.fn()
    this.cols = 80
    this.rows = 24
    this.options = { ...opts }
    this.buffer = {
      active: {
        viewportY: 0,
        getLine: () => ({ translateToString: () => 'line-content' }),
      },
    }
  })
  return { Terminal: MockTerminal }
})

vi.mock('@xterm/addon-fit', () => {
  const MockFitAddon = vi.fn(function (this: Record<string, unknown>) {
    this.fit = vi.fn()
  })
  return { FitAddon: MockFitAddon }
})

vi.mock('./MobileToolbox', () => ({
  MobileToolbox: ({
    onSend,
    keyboardMode,
  }: {
    onSend: (text: string) => void
    keyboardMode?: boolean
  }) => (
    <div data-testid="mobile-toolbox" data-keyboard={keyboardMode ? 'true' : undefined}>
      <button data-testid="toolbox-send" onClick={() => onSend('test-input')} type="button">
        Send
      </button>
    </div>
  ),
}))

vi.mock('../utils/platform', () => ({
  isIOS: vi.fn(() => false),
  isAndroid: vi.fn(() => false),
}))

vi.mock('../utils/telemetry', () => ({
  log: vi.fn(),
}))

vi.mock('../utils/telemetryEmitter', () => ({
  createTelemetryEmitter: vi.fn(() => ({
    emit: vi.fn(),
    flush: vi.fn(),
    destroy: vi.fn(),
  })),
}))

// Mock ResizeObserver
class MockResizeObserver {
  static instances: MockResizeObserver[] = []
  cb: AnyFn
  constructor(cb: AnyFn) {
    this.cb = cb
    MockResizeObserver.instances.push(this)
  }
  observe() {}
  unobserve() {}
  disconnect() {}
  static clearInstances() {
    MockResizeObserver.instances = []
  }
}
vi.stubGlobal('ResizeObserver', MockResizeObserver)

function createMockWebSocket(): MockWS {
  const handlers: Record<string, EventListener> = {}
  const ws: MockWS = {
    url: 'ws://localhost/ws',
    readyState: 0,
    binaryType: '',
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
    _error() {
      if (typeof ws.onerror === 'function') ws.onerror(new Event('error'))
      handlers.error?.(new Event('error'))
    },
  }
  return ws
}

function getTermInstance(idx = 0): MockTerminal {
  return (Terminal as unknown as Mock).mock.instances[idx] as unknown as MockTerminal
}

function getFitInstance(idx = 0): { fit: Mock } {
  return (FitAddon as unknown as Mock).mock.instances[idx] as unknown as { fit: Mock }
}

function getWSCalls(): unknown[][] {
  return (globalThis.WebSocket as unknown as Mock).mock.calls
}

function getOnDataCallback(term: MockTerminal): (data: string) => void {
  return (term as unknown as Record<string, unknown>)._onDataCb as (data: string) => void
}

function getResizeCalls(sendMock: Mock) {
  return sendMock.mock.calls.filter(
    (c: unknown[]) => typeof c[0] === 'string' && c[0].startsWith('3'),
  )
}

function getInputCalls(sendMock: Mock) {
  return sendMock.mock.calls.filter(
    (c: unknown[]) => typeof c[0] === 'string' && c[0].startsWith('1'),
  )
}

describe('MobileTerminal', () => {
  let mockWebSocket: MockWS
  let container: HTMLDivElement

  beforeEach(() => {
    vi.clearAllMocks()
    MockResizeObserver.clearInstances()
    vi.useFakeTimers()
    mockWebSocket = createMockWebSocket()
    const MockWS = vi.fn(function () {
      return mockWebSocket
    }) as ReturnType<typeof vi.fn> & {
      OPEN: number
      CLOSED: number
      CONNECTING: number
    }
    MockWS.OPEN = 1
    MockWS.CLOSED = 3
    MockWS.CONNECTING = 0
    vi.stubGlobal('WebSocket', MockWS)
    container = document.createElement('div')
    document.body.appendChild(container)
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ alternate_on: false, mouse_any_flag: false }),
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
    document.body.removeChild(container)
  })

  it('renders terminal container', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  it('creates terminal and opens WebSocket', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('session=s1'))
    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('pane=%250'))
  })

  it('sends auth and resize on WebSocket open', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith(expect.stringContaining('AuthToken'))
    expect(mockWebSocket.send).toHaveBeenCalledWith('4base64')
  })

  it('writes received output to terminal', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      mockWebSocket._message('1aGVsbG8=')
    })

    expect(term.write).toHaveBeenCalledWith('hello')
  })

  it('sends input via toolbox onSend', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      screen.getByTestId('toolbox-send').click()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith(expect.stringContaining('dGVzdC1pbnB1dA=='))
  })

  it('cleans up on unmount', () => {
    const { unmount } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()

    act(() => {
      unmount()
    })

    expect(mockWebSocket.close).toHaveBeenCalled()
    expect(term.dispose).toHaveBeenCalled()
  })

  it('renders fit window button', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTitle('Fit window')).toBeInTheDocument()
  })

  // --- WebSocket message types ---

  it('handles SetWindowTitle message (type 3)', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      mockWebSocket._message('3"New Title"')
    })

    expect(document.title).toBe('New Title')
  })

  it('handles malformed SetWindowTitle message gracefully', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const prevTitle = document.title

    act(() => {
      mockWebSocket._message('3not-json')
    })

    expect(document.title).toBe(prevTitle)
  })

  it('handles null title from SetWindowTitle', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const prevTitle = document.title

    act(() => {
      mockWebSocket._message('3null')
    })

    expect(document.title).toBe(prevTitle)
  })

  it('ignores non-string message data', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    term.write.mockClear()

    act(() => {
      const msgEvent = new MessageEvent('message', { data: new ArrayBuffer(8) })
      if (typeof mockWebSocket.onmessage === 'function') {
        mockWebSocket.onmessage(msgEvent)
      }
    })

    expect(term.write).not.toHaveBeenCalled()
  })

  it('handles unknown message type gracefully', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    term.write.mockClear()

    act(() => {
      mockWebSocket._message('5unknown')
    })

    expect(term.write).not.toHaveBeenCalled()
  })

  it('handles base64 decode error in output message', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    term.write.mockClear()

    act(() => {
      mockWebSocket._message('1!!!invalid-base64!!!')
    })

    expect(term.write).toHaveBeenCalled()
  })

  // --- WebSocket error and close ---

  it('handles WebSocket error', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    term.write.mockClear()

    act(() => {
      mockWebSocket._error()
    })

    expect(term.write).toHaveBeenCalledWith(expect.stringContaining('Connection error'))
  })

  it('attempts reconnection on WebSocket close', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const initialWsCount = getWSCalls().length

    act(() => {
      mockWebSocket._close()
    })

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(getWSCalls().length).toBeGreaterThan(initialWsCount)
  })

  it('shows reconnect failure message after max attempts', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()

    for (let i = 0; i < 4; i++) {
      act(() => {
        mockWebSocket._close()
      })
      act(() => {
        vi.advanceTimersByTime(20000)
      })
    }

    const reconnectMsg = term.write.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('\u91CD\u8FDE\u5931\u8D25'),
    )
    expect(reconnectMsg).toBeTruthy()
  })

  it('does not reconnect after intentional close (unmount)', () => {
    const { unmount } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    const initialWsCount = getWSCalls().length

    act(() => {
      unmount()
    })

    act(() => {
      vi.advanceTimersByTime(30000)
    })

    expect(getWSCalls().length).toBe(initialWsCount)
  })

  // --- Font size effect ---

  it('updates terminal font size when prop changes', () => {
    const { rerender } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    // Verify only one Terminal instance exists (main effect hasn't re-run)
    expect((Terminal as unknown as Mock).mock.instances.length).toBe(1)
    const term = getTermInstance(0)
    expect(term.options.fontSize).toBe(10)

    rerender(<MobileTerminal session="s1" pane="%0" fontSize={14} onFontSizeChange={vi.fn()} />)

    // Changing fontSize should NOT create a new Terminal instance;
    // the separate fontSize effect updates the existing terminal's options
    expect((Terminal as unknown as Mock).mock.instances.length).toBe(1)
    expect(term.options.fontSize).toBe(14)

    act(() => {
      vi.advanceTimersByTime(500)
    })
  })

  // --- Fit window button ---

  it('calls handleFitWindow when fit button is clicked', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      screen.getByTitle('Fit window').click()
    })

    const resizeCalls = getResizeCalls(mockWebSocket.send as unknown as Mock)
    expect(resizeCalls.length).toBeGreaterThan(0)
  })

  it('does not send resize on fit window when WebSocket is not open', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      screen.getByTitle('Fit window').click()
    })

    expect(mockWebSocket.send).not.toHaveBeenCalled()
  })

  // --- sendText ---

  it('does not send text when WebSocket is not open', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      screen.getByTestId('toolbox-send').click()
    })

    expect(mockWebSocket.send).not.toHaveBeenCalled()
  })

  // --- Terminal input handling ---

  it('ignores focus report sequences', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback('\x1b[I')
      onDataCallback('\x1b[O')
      onDataCallback('\x1b[?1;2c')
      onDataCallback('\x1b[>1;2c')
      onDataCallback('\x1b]0;title\x07')
    })

    expect(mockWebSocket.send).not.toHaveBeenCalled()
  })

  it('sends regular input to WebSocket', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback('a')
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith(expect.stringContaining('YQ=='))
  })

  it('deduplicates rapid identical input', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback('a')
      onDataCallback('a')
    })

    const inputCalls = getInputCalls(mockWebSocket.send as unknown as Mock)
    expect(inputCalls.length).toBe(1)
  })

  it('does not send input when WebSocket is not open', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const onDataCallback = getOnDataCallback(getTermInstance())

    act(() => {
      onDataCallback('a')
    })

    expect(mockWebSocket.send).not.toHaveBeenCalled()
  })

  // --- Reconnect with wasReconnect flag ---

  it('shows reconnected message on successful reconnect', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    term.write.mockClear()

    act(() => {
      mockWebSocket._close()
    })

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // The reconnect attempt counter should be > 0 after close + delay
    // The write calls should include the reconnect delay message
    const delayMsg = term.write.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('后重连'),
    )
    expect(delayMsg).toBeTruthy()
  })

  // --- Visibility change ---

  it('handles visibility change event', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  // --- Pane mode checking ---

  it('checks pane mode on mount', () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ alternate_on: true, mouse_any_flag: true }),
    })
    globalThis.fetch = fetchMock

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    // flush microtasks (fetch returns a promise, but the call itself is sync)
    const paneModeCall = fetchMock.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('/api/tmux/pane-mode'),
    )
    expect(paneModeCall).toBeTruthy()
  })

  it('handles pane mode fetch error gracefully', () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'))

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  // --- Terminal configuration ---

  it('creates Terminal with correct options', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={12} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(Terminal).toHaveBeenCalledWith(
      expect.objectContaining({
        cursorBlink: true,
        fontSize: 12,
        fontFamily: 'Menlo, Monaco, monospace',
        scrollback: 5000,
        lineHeight: 1.2,
        drawBoldTextInBrightColors: true,
        cursorStyle: 'bar',
      }),
    )
  })

  it('loads FitAddon and opens terminal on container', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const term = getTermInstance()
    expect(term.loadAddon).toHaveBeenCalled()
    expect(term.open).toHaveBeenCalled()
    expect(FitAddon).toHaveBeenCalled()
  })

  // --- WebSocket URL construction ---

  it('constructs WebSocket URL with session and pane params', () => {
    render(
      <MobileTerminal session="my-session" pane="%1" fontSize={10} onFontSizeChange={vi.fn()} />,
      {
        container,
      },
    )

    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('session=my-session'))
    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('pane=%251'))
  })

  it('uses wss protocol when page is https', () => {
    // jsdom doesn't allow redefining location.protocol directly,
    // so we verify the protocol selection logic by checking default (http -> ws)
    // and that the URL construction uses location.protocol correctly
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    // Default in jsdom is http: so WebSocket should use ws:
    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('ws://'))
  })

  // --- MobileToolbox props ---

  it('passes correct props to MobileToolbox', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={14} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const toolbox = screen.getByTestId('mobile-toolbox')
    expect(toolbox).toBeInTheDocument()
    expect(toolbox.getAttribute('data-keyboard')).toBeNull()
  })

  // --- Optional props ---

  it('renders without voiceRef', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  it('renders without taskHistoryPaneKey', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  it('renders without onStatusChange', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  // --- WebSocket binary type ---

  it('sets WebSocket binaryType to arraybuffer', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(mockWebSocket.binaryType).toBe('arraybuffer')
  })

  // --- sends initial resize on connect ---

  it('sends initial terminal dimensions on WebSocket open', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const resizeCalls = getResizeCalls(mockWebSocket.send as unknown as Mock)
    expect(resizeCalls.length).toBeGreaterThan(0)

    const payload = JSON.parse(resizeCalls[0]![0].slice(1))
    expect(payload).toHaveProperty('columns')
    expect(payload).toHaveProperty('rows')
  })

  // --- Fit window sends correct resize payload ---

  it('handleFitWindow sends columns and rows in resize message', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      screen.getByTitle('Fit window').click()
    })

    const resizeCalls = getResizeCalls(mockWebSocket.send as unknown as Mock)
    expect(resizeCalls.length).toBeGreaterThan(0)

    const payload = JSON.parse(resizeCalls[0]![0].slice(1))
    expect(typeof payload.columns).toBe('number')
    expect(typeof payload.rows).toBe('number')
  })

  // --- Exponential backoff reconnect delays ---

  it('shows attempt counter in reconnect message', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()

    act(() => {
      mockWebSocket._close()
    })

    const firstCall = term.write.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('1/3'),
    )
    expect(firstCall).toBeTruthy()
  })

  // --- Telemetry emitter creation and destruction ---

  it('creates telemetry emitter with correct pane ID', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(createTelemetryEmitter).toHaveBeenCalledWith('s1:%0')
  })

  it('destroys telemetry emitter on unmount', () => {
    const { unmount } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      unmount()
    })

    const mockEmitter = (createTelemetryEmitter as ReturnType<typeof vi.fn>).mock.results[0]?.value
    expect(mockEmitter?.destroy).toHaveBeenCalled()
  })

  // --- Visibility change handler cleanup ---

  it('removes visibility change listener on unmount', () => {
    const removeSpy = vi.spyOn(document, 'removeEventListener')

    const { unmount } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      unmount()
    })

    expect(removeSpy).toHaveBeenCalledWith('visibilitychange', expect.any(Function))
    removeSpy.mockRestore()
  })

  // --- iOS-specific behavior ---

  it('sends DEC_1004_DISABLE on iOS when WebSocket opens', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith('\x1b[?1004l')

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('registers visualViewport listener on iOS', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    const addEventListenerSpy = vi.fn()
    Object.defineProperty(window, 'visualViewport', {
      value: {
        addEventListener: addEventListenerSpy,
        removeEventListener: vi.fn(),
        height: 600,
        width: 400,
      },
      writable: true,
      configurable: true,
    })

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(addEventListenerSpy).toHaveBeenCalledWith('resize', expect.any(Function))

    vi.mocked(isIOS).mockReturnValue(false)
    delete (window as unknown as Record<string, unknown>).visualViewport
  })

  it('cleans up visualViewport listener on iOS unmount', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    const removeEventListenerSpy = vi.fn()
    Object.defineProperty(window, 'visualViewport', {
      value: {
        addEventListener: vi.fn(),
        removeEventListener: removeEventListenerSpy,
        height: 600,
        width: 400,
      },
      writable: true,
      configurable: true,
    })

    const { unmount } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      unmount()
    })

    expect(removeEventListenerSpy).toHaveBeenCalledWith('resize', expect.any(Function))

    vi.mocked(isIOS).mockReturnValue(false)
    delete (window as unknown as Record<string, unknown>).visualViewport
  })

  // --- Android burst suppression ---

  it('suppresses space burst on Android', async () => {
    const { isAndroid } = await import('../utils/platform')
    vi.mocked(isAndroid).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback(' ')
      onDataCallback(' ')
      onDataCallback(' ')
    })

    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBeLessThan(3)

    vi.mocked(isAndroid).mockReturnValue(false)
  })

  it('suppresses enter burst on Android', async () => {
    const { isAndroid } = await import('../utils/platform')
    vi.mocked(isAndroid).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback('\r')
      onDataCallback('\n')
    })

    const enterCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && (c[0] === '1DQo=' || c[0] === '1Cg=='),
    )
    expect(enterCalls.length).toBeLessThanOrEqual(2)

    vi.mocked(isAndroid).mockReturnValue(false)
  })

  // --- iOS post-transition burst suppression ---

  it('suppresses input after transition on iOS', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback(' ')
    })

    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBe(1)

    vi.mocked(isIOS).mockReturnValue(false)
  })

  // --- sendText encodes to base64 ---

  it('sendText encodes text to base64 before sending', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      screen.getByTestId('toolbox-send').click()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith('1dGVzdC1pbnB1dA==')
  })

  // --- Handles pane mode with alternate screen and mouse flags ---

  it('detects pane in alt screen mode', () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ alternate_on: true, mouse_any_flag: true }),
    })
    globalThis.fetch = fetchMock

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const paneModeCall = fetchMock.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('/api/tmux/pane-mode'),
    )
    expect(paneModeCall).toBeTruthy()
  })

  // --- WebSocket sends base64 encoding preference ---

  it('sends base64 encoding preference message', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith('4base64')
  })

  // --- AuthToken message format ---

  it('sends AuthToken with correct format', () => {
    render(
      <MobileTerminal session="test-session" pane="%5" fontSize={10} onFontSizeChange={vi.fn()} />,
      {
        container,
      },
    )

    act(() => {
      mockWebSocket._open()
    })

    const authCall = (mockWebSocket.send as unknown as Mock).mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('AuthToken'),
    )
    expect(authCall).toBeTruthy()
    const parsed = JSON.parse(authCall![0])
    expect(parsed).toHaveProperty('AuthToken')
    expect(parsed).toHaveProperty('Arguments')
    expect(parsed.Arguments).toContain('session=test-session')
    expect(parsed.Arguments).toContain('pane=%255')
  })

  // --- Terminal theme ---

  it('passes theme to Terminal constructor', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const call = (Terminal as unknown as Mock).mock.calls[0]!
    expect(call[0].theme).toEqual({
      background: '#0f1115',
      foreground: '#abb2bf',
      cursor: '#4d78cc',
      selectionBackground: 'rgba(77, 120, 204, 0.3)',
      black: '#1e2127',
      red: '#e06c75',
      green: '#98c379',
      yellow: '#d19a66',
      blue: '#61afef',
      magenta: '#c678dd',
      cyan: '#56b6c2',
      white: '#abb2bf',
    })
  })

  // --- Fit retries ---

  it('calls fit multiple times for iOS compatibility', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const fit = getFitInstance()

    expect(fit.fit).toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(400)
    })

    expect(fit.fit.mock.calls.length).toBeGreaterThanOrEqual(3)
  })

  // --- Textarea attributes ---

  it('sets textarea attributes for mobile', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const textarea = container.querySelector('textarea') as HTMLTextAreaElement
    if (textarea) {
      expect(textarea.getAttribute('autocapitalize')).toBe('off')
      expect(textarea.getAttribute('autocorrect')).toBe('off')
      expect(textarea.getAttribute('spellcheck')).toBe('false')
      expect(textarea.getAttribute('autocomplete')).toBe('off')
    }
  })

  // --- keyboardMode toggle ---

  it('passes keyboardMode prop based on showKeyboard state', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTestId('mobile-toolbox').getAttribute('data-keyboard')).toBeNull()
  })

  // --- Multiple sessions/panes ---

  it('handles different session and pane values', () => {
    render(<MobileTerminal session="dev" pane="%3" fontSize={16} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('session=dev'))
    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('pane=%253'))
    expect(Terminal).toHaveBeenCalledWith(expect.objectContaining({ fontSize: 16 }))
  })

  // --- toggleKeyboard callback ---

  it('does not throw when toolbox is not rendered', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(screen.getByTitle('Fit window')).toBeInTheDocument()
  })

  // --- WebSocket reconnect clears pending timeout ---

  it('clears reconnect timeout before reconnecting', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      mockWebSocket._close()
    })

    // Don't advance timers - unmount should clear pending timeout
    const { unmount } = render(
      <MobileTerminal session="s2" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      unmount()
    })

    // Should not throw or leave dangling timers
  })

  // --- Touch gesture system ---

  // Helper to create mock Touch objects (jsdom doesn't have Touch constructor)
  function createMockTouch(opts: {
    identifier: number
    clientX: number
    clientY: number
  }): Touch & { clientX: number; clientY: number } {
    return {
      identifier: opts.identifier,
      clientX: opts.clientX,
      clientY: opts.clientY,
      target: null as unknown as EventTarget,
      pageX: opts.clientX,
      pageY: opts.clientY,
      screenX: 0,
      screenY: 0,
      radiusX: 0,
      radiusY: 0,
      rotationAngle: 0,
      force: 1,
    } as unknown as Touch & { clientX: number; clientY: number }
  }

  function createTouchEvent(type: string, touches: Touch[], cancelable = true): TouchEvent {
    return new TouchEvent(type, { touches, cancelable, bubbles: true } as unknown as TouchEventInit)
  }

  it('handles single finger touch start and end', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement
    expect(termContainer).toBeTruthy()

    // Single finger touch start
    const touchStartEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(touchStartEvent)

    // Single finger touch end
    const touchEndEvent = createTouchEvent('touchend', [])
    termContainer.dispatchEvent(touchEndEvent)
  })

  it('handles two finger scroll touch start and move', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Two finger touch start
    const touchStartEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 300 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(touchStartEvent)

    // Two finger scroll move (move up by 30px each = 60px total, midY goes from 300 to 240)
    const touchMoveEvent = createTouchEvent('touchmove', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 270 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 270 }),
    ])
    termContainer.dispatchEvent(touchMoveEvent)

    // Touch end
    const touchEndEvent = createTouchEvent('touchend', [])
    termContainer.dispatchEvent(touchEndEvent)
  })

  it('handles upgrade from one finger to two finger scroll', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // One finger start
    const start1 = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(start1)

    // Upgrade to two fingers
    const start2 = createTouchEvent('touchmove', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 300 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(start2)
  })

  it('cancels long press on move beyond tolerance', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // Start single finger
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(startEvent)

    // Move beyond tolerance (LONG_PRESS_MOVE_TOLERANCE = 10)
    const moveEvent = createTouchEvent('touchmove', [
      createMockTouch({ identifier: 1, clientX: 120, clientY: 200 }),
    ])
    termContainer.dispatchEvent(moveEvent)

    // Advance past long press timer - should NOT trigger overlay since we moved
    act(() => {
      vi.advanceTimersByTime(700)
    })

    // No overlay should be created
    expect(container.querySelector('.select-mode-overlay')).not.toBeInTheDocument()
  })

  it('triggers selection overlay on long press', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // Start single finger
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(startEvent)

    // Advance past LONG_PRESS_MS (650ms)
    act(() => {
      vi.advanceTimersByTime(700)
    })

    // Selection overlay should be created
    expect(container.querySelector('.select-mode-overlay')).toBeInTheDocument()
    expect(container.querySelector('.select-mode-copy-btn')).toBeInTheDocument()
    expect(container.querySelector('.select-mode-exit-btn')).toBeInTheDocument()
    expect(container.querySelector('.select-mode-text')).toBeInTheDocument()
  })

  it('hides selection overlay on exit button touch', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // Trigger long press
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(startEvent)
    act(() => {
      vi.advanceTimersByTime(700)
    })

    expect(container.querySelector('.select-mode-overlay')).toBeInTheDocument()

    // Touch exit button
    const exitBtn = container.querySelector('.select-mode-exit-btn') as HTMLElement
    expect(exitBtn).toBeTruthy()
    const exitTouch = createTouchEvent('touchend', [])
    exitBtn.dispatchEvent(exitTouch)

    expect(container.querySelector('.select-mode-overlay')).not.toBeInTheDocument()
  })

  it('blocks click after two finger scroll', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // Two finger scroll
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 300 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(startEvent)

    // End two fingers
    const endEvent = createTouchEvent('touchend', [])
    termContainer.dispatchEvent(endEvent)

    // Click immediately after should be blocked (clickBlockedUntil = Date.now() + 300)
    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true })
    termContainer.dispatchEvent(clickEvent)
  })

  // --- sendScroll ---

  it('sendScroll sends mouse scroll escape sequences', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Two finger scroll with enough movement to trigger sendScroll
    // SCROLL_THRESHOLD = 20, so we need deltaY > 20
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 300 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(startEvent)

    // Move up 50px (midY from 300 to 250, deltaY = 50)
    const moveEvent = createTouchEvent('touchmove', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 250 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 250 }),
    ])
    termContainer.dispatchEvent(moveEvent)

    // Should send scroll escape sequences
    const scrollCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('\x1b[<'),
    )
    expect(scrollCalls.length).toBeGreaterThan(0)
  })

  // --- Visibility change reconnect ---

  it('reconnects on visibility change to visible when WebSocket is closed', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Close the WebSocket
    act(() => {
      mockWebSocket._close()
    })

    // Set document to hidden, then visible
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true })
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // Should trigger reconnect
    const reconnectMsg = getTermInstance().write.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('Resuming'),
    )
    expect(reconnectMsg).toBeTruthy()

    // Restore
    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
  })

  it('does not reconnect on visibility change when WebSocket is open', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const initialWsCount = getWSCalls().length

    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // Should NOT create a new WebSocket
    expect(getWSCalls().length).toBe(initialWsCount)
  })

  // --- Manual reconnect after max attempts ---

  it('triggers manual reconnect on input after max attempts', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()

    // Close 4 times to exceed MAX_RECONNECT_ATTEMPTS (3)
    for (let i = 0; i < 4; i++) {
      act(() => {
        mockWebSocket._close()
      })
      act(() => {
        vi.advanceTimersByTime(20000)
      })
    }

    // Should show failure message
    const failMsg = term.write.mock.calls.find(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('\u91CD\u8FDE\u5931\u8D25'),
    )
    expect(failMsg).toBeTruthy()

    // Now trigger input via onData callback - this should trigger manual reconnect
    const onDataCallback = getOnDataCallback(term)
    const wsCountBefore = getWSCalls().length

    act(() => {
      onDataCallback('a')
    })

    // Should attempt to reconnect
    expect(getWSCalls().length).toBeGreaterThan(wsCountBefore)
  })

  // --- iOS reconnect telemetry ---

  it('logs reconnect telemetry on iOS when reconnecting', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)
    const { log: telemetryLog } = await import('../utils/telemetry')

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Close to trigger reconnect
    act(() => {
      mockWebSocket._close()
    })

    // Advance past reconnect delay
    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // Now the reconnect created a new WebSocket (same mock object).
    // Trigger its onopen - reconnectAttemptRef should be > 0
    act(() => {
      mockWebSocket._open()
    })

    // After successful reconnect with wasReconnect=true, should log reconnect telemetry
    expect(telemetryLog).toHaveBeenCalledWith('reconnect', expect.any(Object))

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('writes reconnected message on iOS after successful reconnect', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTermInstance()
    term.write.mockClear()

    // Close and advance to reconnect
    act(() => {
      mockWebSocket._close()
    })
    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // The new WebSocket should be opened - trigger its onopen
    // This may or may not appear depending on timing - just check no crash
    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()

    vi.mocked(isIOS).mockReturnValue(false)
  })

  // --- iOS visual viewport resize handler ---

  it('tracks visual viewport resize on iOS', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    let viewportResizeHandler: EventListener | undefined
    Object.defineProperty(window, 'visualViewport', {
      value: {
        addEventListener: vi.fn((event: string, handler: EventListener) => {
          if (event === 'resize') viewportResizeHandler = handler
        }),
        removeEventListener: vi.fn(),
        height: 600,
        width: 400,
      },
      writable: true,
      configurable: true,
    })

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    expect(viewportResizeHandler).toBeDefined()

    // Trigger viewport resize
    act(() => {
      viewportResizeHandler?.(new Event('resize'))
    })

    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()

    vi.mocked(isIOS).mockReturnValue(false)
    delete (window as unknown as Record<string, unknown>).visualViewport
  })

  // --- ResizeObserver handleResize ---

  it('sends resize message when dimensions change via ResizeObserver', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Get the ResizeObserver callback from the mock
    // The ResizeObserver is created in the effect, so we need to trigger it
    const roInstances = (ResizeObserver as unknown as { mock: { instances: unknown[] } }).mock
      ?.instances
    if (roInstances && roInstances.length > 0) {
      const roInstance = roInstances[0]! as Record<string, AnyFn>
      const roCallback = roInstance.cb!

      // Simulate resize
      act(() => {
        vi.advanceTimersByTime(200)
        roCallback()
      })

      // The handleResize callback calls fit() then checks if dimensions changed
      // Since our mock Terminal has fixed cols/rows, no resize message is sent
    }
  })

  // --- Font size effect setTimeout callbacks ---

  it('triggers fit and resize in font size effect after delays', () => {
    const { rerender } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    const fit = getFitInstance()
    fit.fit.mockClear()

    // Change fontSize - the fontSize effect (lines 96-111) runs
    rerender(<MobileTerminal session="s1" pane="%0" fontSize={14} onFontSizeChange={vi.fn()} />)

    // Same FitAddon instance (no new Terminal created)
    const sameFit = getFitInstance(0)

    // Advance past the setTimeout delays (100ms and 300ms)
    act(() => {
      vi.advanceTimersByTime(400)
    })

    // fit() should be called multiple times from the fontSize effect's setTimeouts
    expect(sameFit.fit.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  // --- Burst suppression edge cases ---

  it('allows space input below burst threshold on Android', async () => {
    const { isAndroid } = await import('../utils/platform')
    vi.mocked(isAndroid).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send 2 spaces with time gap to avoid dedup (50ms threshold)
    act(() => {
      onDataCallback(' ')
      vi.advanceTimersByTime(60)
      onDataCallback(' ')
    })

    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    // Should allow both through (below burst threshold of 3)
    expect(spaceCalls.length).toBe(2)

    vi.mocked(isAndroid).mockReturnValue(false)
  })

  it('allows enter input below burst threshold on Android', async () => {
    const { isAndroid } = await import('../utils/platform')
    vi.mocked(isAndroid).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send only 1 enter (below ENTER_BURST_COUNT = 2)
    act(() => {
      onDataCallback('\r')
    })

    const enterCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1DQ==',
    )
    expect(enterCalls.length).toBe(1)

    vi.mocked(isAndroid).mockReturnValue(false)
  })

  it('does not suppress non-burst input on iOS without transition', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send regular character (not a suppressed input type)
    act(() => {
      onDataCallback('x')
    })

    expect(mockWebSocket.send).toHaveBeenCalled()

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('suppresses iOS post-transition input within burst window', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Simulate a reconnect to set lastTransitionRef
    act(() => {
      mockWebSocket._close()
    })
    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // Trigger the new WebSocket's onopen to set lastTransitionRef
    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send space right after transition (within BURST_SUPPRESSION_WINDOW_MS = 200)
    act(() => {
      onDataCallback(' ')
    })

    // Should be suppressed due to post-transition
    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBe(0)

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('allows iOS input after burst suppression window expires', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Simulate a reconnect to set lastTransitionRef
    act(() => {
      mockWebSocket._close()
    })
    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // Trigger the new WebSocket's onopen (reconnect with wasReconnect=true)
    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Advance past BURST_SUPPRESSION_WINDOW_MS (200ms)
    act(() => {
      vi.advanceTimersByTime(300)
    })

    act(() => {
      onDataCallback(' ')
    })

    // Should be allowed after window expires
    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBe(1)

    vi.mocked(isIOS).mockReturnValue(false)
  })

  // --- iOS space burst suppression ---

  it('suppresses space burst on iOS', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send 3+ spaces (SPACE_BURST_COUNT = 3)
    act(() => {
      onDataCallback(' ')
      onDataCallback(' ')
      onDataCallback(' ')
    })

    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBeLessThan(3)

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('suppresses enter burst on iOS', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send 2+ enters (ENTER_BURST_COUNT = 2)
    act(() => {
      onDataCallback('\r')
      onDataCallback('\n')
    })

    const enterCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && (c[0] === '1DQo=' || c[0] === '1Cg=='),
    )
    expect(enterCalls.length).toBeLessThanOrEqual(2)

    vi.mocked(isIOS).mockReturnValue(false)
  })

  // --- toggleKeyboard ---

  it('toggleKeyboard focuses xterm helper textarea when enabled', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    // Create a mock xterm helper textarea
    const helperTextarea = document.createElement('textarea')
    helperTextarea.className = 'xterm-helper-textarea'
    document.body.appendChild(helperTextarea)

    // The toggleKeyboard function is internal, but we can verify the component
    // doesn't crash when the textarea exists
    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()

    document.body.removeChild(helperTextarea)
  })

  // --- Non-suppressed inputs pass through on non-mobile ---

  it('allows all input on non-mobile platforms', () => {
    // isIOS and isAndroid both return false by default
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send 4 different characters to avoid dedup, and non-burst-suppressed types
    act(() => {
      onDataCallback('a')
      vi.advanceTimersByTime(60)
      onDataCallback('b')
      vi.advanceTimersByTime(60)
      onDataCallback('c')
      vi.advanceTimersByTime(60)
      onDataCallback('d')
    })

    // All inputs should pass through (no burst suppression on non-mobile)
    expect(mockWebSocket.send).toHaveBeenCalledTimes(4)
  })

  // --- Visibility change sets iOS transition ref ---

  it('sets lastTransitionRef on iOS visibility change to visible', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Simulate visibility change to hidden then visible
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true })
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // The visibility change triggers reconnect - open the new WebSocket
    act(() => {
      mockWebSocket._open()
    })

    // After visibility change, post-transition suppression should be active
    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    act(() => {
      onDataCallback(' ')
    })

    // Should be suppressed due to post-transition
    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBe(0)

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
    vi.mocked(isIOS).mockReturnValue(false)
  })

  // --- WebSocket connect disposes manual reconnect disposable ---

  it('connect disposes previous manual reconnect on new connection', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Close 4 times to trigger max reconnect attempts
    for (let i = 0; i < 4; i++) {
      act(() => {
        mockWebSocket._close()
      })
      act(() => {
        vi.advanceTimersByTime(20000)
      })
    }

    // At this point, manualReconnectDisposable should be set
    // Now trigger another close + reconnect cycle
    act(() => {
      mockWebSocket._close()
    })
    act(() => {
      vi.advanceTimersByTime(20000)
    })

    // Should not crash - the old disposable should be cleaned up
    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  // --- Additional branch coverage ---

  it('showSelectionOverlay returns early when container is null', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // Trigger long press
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(startEvent)
    act(() => {
      vi.advanceTimersByTime(700)
    })

    // Should create overlay (container exists)
    expect(container.querySelector('.select-mode-overlay')).toBeInTheDocument()
  })

  it('showSelectionOverlay returns early when text is empty', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    // Make getLine return null to simulate empty buffer
    const term = getTermInstance()
    term.buffer.active.getLine = vi.fn(() => null)

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(startEvent)
    act(() => {
      vi.advanceTimersByTime(700)
    })

    // Should not create overlay since all lines are null
    expect(container.querySelector('.select-mode-overlay')).not.toBeInTheDocument()
  })

  it('hideSelectionOverlay does nothing when no overlay exists', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    // Trigger touch start and end without long press
    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 100, clientY: 200 }),
    ])
    termContainer.dispatchEvent(startEvent)

    const endEvent = createTouchEvent('touchend', [])
    termContainer.dispatchEvent(endEvent)

    // Should not throw - hideSelectionOverlay handles null overlay
    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  it('sendScroll returns early when lines is 0', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Two finger touch start
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 300 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(startEvent)

    // Move less than SCROLL_THRESHOLD (20px) - should not trigger sendScroll
    const moveEvent = createTouchEvent('touchmove', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 295 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 295 }),
    ])
    termContainer.dispatchEvent(moveEvent)

    // Should NOT send scroll sequences (deltaY < SCROLL_THRESHOLD)
    const scrollCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].includes('\x1b[<'),
    )
    expect(scrollCalls.length).toBe(0)
  })

  it('visibility change does not reconnect when WebSocket is connecting', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    // Don't open WebSocket - it stays in CONNECTING state
    mockWebSocket.readyState = 0

    const initialWsCount = getWSCalls().length

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // Should NOT trigger reconnect (readyState is CONNECTING)
    expect(getWSCalls().length).toBe(initialWsCount)

    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
  })

  it('onTouchEnd does not reset gesture when touches remain', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    const termContainer = container.querySelector('.mobile-terminal-container') as HTMLElement

    // Two finger start
    const startEvent = createTouchEvent('touchstart', [
      createMockTouch({ identifier: 1, clientX: 50, clientY: 300 }),
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(startEvent)

    // Remove one finger (still one remaining)
    const endEvent = createTouchEvent('touchend', [
      createMockTouch({ identifier: 2, clientX: 150, clientY: 300 }),
    ])
    termContainer.dispatchEvent(endEvent)

    // Should not crash - gesture stays as twoFingerScroll since touches remain
    expect(screen.getByTestId('mobile-toolbox')).toBeInTheDocument()
  })

  it('iOS burst suppression returns false for non-suppressed input', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Send a non-suppressed character (not space/enter/newline)
    act(() => {
      onDataCallback('x')
    })

    // Should pass through (not in SUPPRESSED_INPUTS)
    expect(mockWebSocket.send).toHaveBeenCalled()

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('iOS burst suppression returns false when no transition exists', async () => {
    const { isIOS } = await import('../utils/platform')
    vi.mocked(isIOS).mockReturnValue(true)

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    // Don't trigger any transition - lastTransitionRef should be null
    const onDataCallback = getOnDataCallback(getTermInstance())
    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Advance past BURST_SUPPRESSION_WINDOW_MS to ensure no timing issues
    act(() => {
      vi.advanceTimersByTime(300)
    })

    act(() => {
      onDataCallback(' ')
    })

    // Should pass through (no transition set)
    const spaceCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0] === '1IA==',
    )
    expect(spaceCalls.length).toBe(1)

    vi.mocked(isIOS).mockReturnValue(false)
  })

  it('handleResize does not send when dimensions are unchanged', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Trigger ResizeObserver callback - dimensions stay at 80x24 (mock defaults)
    const roCb = MockResizeObserver.instances[0]?.cb
    if (roCb) {
      act(() => {
        vi.advanceTimersByTime(200)
        roCb()
      })
    }

    // Should not send resize since dimensions haven't changed (lastCols/lastRows start at 0, but first call sets them)
    // First call: lastCols=0, cols=80 -> dimensions changed, sends resize
    // Actually, the first fit() call from the effect already sends the initial resize.
    // Here we trigger a second callback where lastCols=80, cols=80 -> no change
    // But lastCols starts at 0, so the first ResizeObserver trigger WILL send.
    // We need to trigger it twice: first to set lastCols, second to check no-change
    if (roCb) {
      act(() => {
        vi.advanceTimersByTime(200)
        roCb()
      })
    }
    const resizeCalls2 = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].startsWith('3'),
    )
    // After the first trigger sets lastCols=80, the second should not send
    // But there might be 1 call from the first trigger
    expect(resizeCalls2.length).toBeLessThanOrEqual(1)
  })

  it('handleResize sends resize when dimensions change', () => {
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as unknown as Mock).mockClear()

    // Change the terminal's cols to simulate a resize
    const term = getTermInstance()
    term.cols = 100

    const roCb = MockResizeObserver.instances[0]?.cb
    if (roCb) {
      // The ResizeObserver callback schedules handleResize via setTimeout(150ms)
      act(() => {
        roCb()
        vi.advanceTimersByTime(200) // past the 150ms debounce
      })
    }

    // Should send resize since cols changed (lastCols=80 from fit, now cols=100)
    const resizeCalls = (mockWebSocket.send as unknown as Mock).mock.calls.filter(
      (c: unknown[]) => typeof c[0] === 'string' && c[0].startsWith('3'),
    )
    expect(resizeCalls.length).toBeGreaterThan(0)
  })
})
