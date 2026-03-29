import { act, render, screen } from '@testing-library/react'
import { Terminal } from '@xterm/xterm'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { TerminalTab } from '../components/TerminalTab'

type AnyFn = (...args: unknown[]) => unknown

interface MockWS extends Record<string, unknown> {
  _open: () => void
  _message: (data: string) => void
  _close: () => void
  _error: () => void
}

// Mock xterm.js modules — use function syntax (not arrow) so they work as constructors
vi.mock('@xterm/xterm', () => {
  const MockTerminal = vi.fn(function (this: Record<string, unknown>) {
    this.open = vi.fn()
    this.loadAddon = vi.fn()
    this.write = vi.fn()
    this.dispose = vi.fn()
    this.onData = vi.fn()
    this.onResize = vi.fn()
    this.cols = 80
    this.rows = 24
  })
  return { Terminal: MockTerminal }
})

vi.mock('@xterm/addon-fit', () => {
  const MockFitAddon = vi.fn(function (this: Record<string, unknown>) {
    this.fit = vi.fn()
  })
  return { FitAddon: MockFitAddon }
})

vi.mock('@xterm/addon-web-links', () => {
  const MockWebLinksAddon = vi.fn(function () {})
  return { WebLinksAddon: MockWebLinksAddon }
})

// Helper to create a mock WebSocket
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
    // Simulate open
    _open() {
      ws.readyState = 1
      if (typeof ws.onopen === 'function') ws.onopen(new Event('open'))
      handlers.open?.(new Event('open'))
    },
    // Simulate incoming message
    _message(data: string) {
      if (typeof ws.onmessage === 'function') ws.onmessage(new MessageEvent('message', { data }))
      handlers.message?.(new MessageEvent('message', { data }))
    },
    // Simulate close
    _close() {
      ws.readyState = 3
      if (typeof ws.onclose === 'function') ws.onclose(new CloseEvent('close'))
      handlers.close?.(new CloseEvent('close'))
    },
    // Simulate error
    _error() {
      if (typeof ws.onerror === 'function') ws.onerror(new Event('error'))
      handlers.error?.(new Event('error'))
    },
  }
  return ws
}

// Helper to get the last Terminal mock instance
function getTerminalMock() {
  const MockedTerminal = vi.mocked(Terminal)
  const instances = MockedTerminal.mock.instances
  const last = instances[instances.length - 1]!
  return last as unknown as Record<string, AnyFn>
}

describe('TerminalTab', () => {
  let mockWebSocket: MockWS
  let container: HTMLDivElement

  beforeEach(() => {
    vi.clearAllMocks()
    mockWebSocket = createMockWebSocket()
    const MockWS = vi.fn(function () {
      return mockWebSocket
    }) as ReturnType<typeof vi.fn> & { OPEN: number; CLOSED: number }
    MockWS.OPEN = 1
    MockWS.CLOSED = 3
    vi.stubGlobal('WebSocket', MockWS)
    container = document.createElement('div')
    document.body.appendChild(container)
  })

  afterEach(() => {
    vi.clearAllMocks()
    document.body.removeChild(container)
  })

  it('renders status bar and terminal container', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    expect(screen.getByText(/connecting|connected|disconnected/)).toBeInTheDocument()
  })

  it('shows connecting status initially', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })
    expect(screen.getByText('● connecting')).toBeInTheDocument()
  })

  it('shows connected status when WebSocket opens', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    expect(screen.getByText('● connected')).toBeInTheDocument()
  })

  it('shows disconnected status when WebSocket closes', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })
    act(() => {
      mockWebSocket._close()
    })

    expect(screen.getByText('● disconnected')).toBeInTheDocument()
  })

  it('sends auth init and setEncoding on WebSocket open', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    // Should have sent init message and '4base64'
    expect(mockWebSocket.send).toHaveBeenCalledTimes(2)
    expect(mockWebSocket.send).toHaveBeenCalledWith(expect.stringContaining('AuthToken'))
    expect(mockWebSocket.send).toHaveBeenCalledWith('4base64')
  })

  it('displays session info in status bar', () => {
    render(<TerminalTab session="my-session" pane="1" />, { container })
    expect(screen.getByText('my-session:1')).toBeInTheDocument()
  })

  it('opens WebSocket with correct URL params', () => {
    render(<TerminalTab session="test-session" pane="pane-0" />, { container })

    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('session=test-session'))
    expect(WebSocket).toHaveBeenCalledWith(expect.stringContaining('pane=pane-0'))
  })

  it('receives terminal output and writes to xterm', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTerminalMock()

    // Send base64-encoded output: "hello" -> aGVsbG8=
    act(() => {
      mockWebSocket._message('1aGVsbG8=')
    })

    expect(term.write).toHaveBeenCalledWith('hello')
  })

  it('updates document title on SetWindowTitle message', () => {
    const originalTitle = document.title
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    act(() => {
      mockWebSocket._message('3"my terminal title"')
    })

    expect(document.title).toBe('my terminal title')
    document.title = originalTitle
  })

  it('sends keystrokes as base64 type 1 messages', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTerminalMock()
    const onDataMock = term.onData as ReturnType<typeof vi.fn>
    const onDataCalls = onDataMock.mock.calls

    // Verify the callback was registered
    expect(onDataCalls.length).toBeGreaterThanOrEqual(1)
    const onDataCallback = onDataCalls[0]![0] as (data: string) => void
    expect(typeof onDataCallback).toBe('function')

    // Clear send history so we only check for the keystroke message
    ;(vi.mocked(mockWebSocket.send) as ReturnType<typeof vi.fn>).mockClear()

    act(() => {
      onDataCallback('a')
    })

    // Should send '1' + btoa('a') = '1YQ=='
    expect(mockWebSocket.send).toHaveBeenCalledWith('1YQ==')
  })

  it('sends resize as type 3 JSON messages', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTerminalMock()
    const onResizeMock = term.onResize as ReturnType<typeof vi.fn>
    const onResizeCalls = onResizeMock.mock.calls

    expect(onResizeCalls.length).toBeGreaterThanOrEqual(1)
    const onResizeCallback = onResizeCalls[0]![0] as (data: { cols: number; rows: number }) => void

    act(() => {
      onResizeCallback({ cols: 120, rows: 40 })
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith(
      `3${JSON.stringify({ columns: 120, rows: 40 })}`,
    )
  })

  it('shows error status on ws onerror', () => {
    render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._error()
    })

    expect(screen.getByText('● disconnected')).toBeInTheDocument()
  })

  it('cleans up on unmount', () => {
    const { unmount } = render(<TerminalTab session="ttyweb" pane="" />, { container })

    act(() => {
      mockWebSocket._open()
    })

    const term = getTerminalMock()

    act(() => {
      unmount()
    })

    expect(mockWebSocket.close).toHaveBeenCalled()
    expect(term.dispose).toHaveBeenCalled()
  })
})
