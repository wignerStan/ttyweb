import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { TerminalTab } from './TerminalTab'

vi.mock('../hooks/useWebTTY', () => ({
  useWebTTY: () => ({
    status: 'connected' as const,
    sendText: vi.fn(),
    sendResize: vi.fn(),
    wsRef: { current: null },
  }),
}))

vi.mock('@xterm/xterm', () => ({
  Terminal: vi.fn().mockImplementation(function () {
    return {
      loadAddon: vi.fn(),
      open: vi.fn(),
      dispose: vi.fn(),
      onData: vi.fn(),
      onResize: vi.fn(),
      cols: 80,
      rows: 24,
    }
  }),
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: vi.fn().mockImplementation(function () {
    return {
      fit: vi.fn(),
      dispose: vi.fn(),
    }
  }),
}))

vi.mock('@xterm/addon-web-links', () => ({
  WebLinksAddon: vi.fn(),
}))

describe('TerminalTab', () => {
  it('renders status bar with connected state', () => {
    const { container } = render(<TerminalTab session="test" pane="0" />)
    expect(container).toMatchSnapshot()
  })

  it('shows session name in status bar', () => {
    render(<TerminalTab session="mysession" pane="1" />)
    expect(screen.getByText(/mysession:1/)).toBeInTheDocument()
  })
})
