import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MobileTerminal } from './MobileTerminal'

type AnyFn = (...args: unknown[]) => unknown

interface MockWS extends Record<string, unknown> {
  _open: () => void
  _message: (data: string) => void
  _close: () => void
  _error: () => void
}

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
    this.options = { fontSize: 10 }
    this.buffer = {
      active: {
        viewportY: 0,
        getLine: () => ({ translateToString: () => '' }),
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

describe('MobileTerminal', () => {
  let mockWebSocket: MockWS
  let container: HTMLDivElement

  beforeEach(() => {
    vi.clearAllMocks()
    mockWebSocket = createMockWebSocket()
    const MockWS = vi.fn(function () {
      return mockWebSocket
    }) as ReturnType<typeof vi.fn> & {
      OPEN: number
      CLOSED: number
    }
    MockWS.OPEN = 1
    MockWS.CLOSED = 3
    vi.stubGlobal('WebSocket', MockWS)
    container = document.createElement('div')
    document.body.appendChild(container)
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ alternate_on: false, mouse_any_flag: false }),
    })
  })

  afterEach(() => {
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

  it('sends auth and resize on WebSocket open', async () => {
    const { act } = await import('@testing-library/react')
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith(expect.stringContaining('AuthToken'))
    expect(mockWebSocket.send).toHaveBeenCalledWith('4base64')
  })

  it('writes received output to terminal', async () => {
    const { act } = await import('@testing-library/react')
    const { Terminal } = await import('@xterm/xterm')

    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    const term = Terminal.mock.instances[0] as unknown as Record<string, AnyFn>
    ;(mockWebSocket.send as ReturnType<typeof vi.fn>).mockClear()

    act(() => {
      mockWebSocket._message('1aGVsbG8=')
    })

    expect(term.write).toHaveBeenCalledWith('hello')
  })

  it('sends input via toolbox onSend', async () => {
    const { act } = await import('@testing-library/react')
    render(<MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />, {
      container,
    })

    act(() => {
      mockWebSocket._open()
    })

    ;(mockWebSocket.send as ReturnType<typeof vi.fn>).mockClear()

    await act(async () => {
      screen.getByTestId('toolbox-send').click()
    })

    expect(mockWebSocket.send).toHaveBeenCalledWith(expect.stringContaining('dGVzdC1pbnB1dA=='))
  })

  it('cleans up on unmount', async () => {
    const { act } = await import('@testing-library/react')
    const { Terminal } = await import('@xterm/xterm')

    const { unmount } = render(
      <MobileTerminal session="s1" pane="%0" fontSize={10} onFontSizeChange={vi.fn()} />,
      { container },
    )

    act(() => {
      mockWebSocket._open()
    })

    const term = Terminal.mock.instances[0] as unknown as Record<string, AnyFn>

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
})
