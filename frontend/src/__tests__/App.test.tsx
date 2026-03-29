import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'

// Mock TerminalTab to avoid xterm.js in unit tests
vi.mock('../components/TerminalTab', () => ({
  TerminalTab: ({ session, pane }: { session: string; pane: string }) => (
    <div data-testid="terminal-tab">
      Terminal: {session}
      {pane ? `:${pane}` : ''}
    </div>
  ),
}))

// Mock Sidebar
vi.mock('../components/Sidebar', () => ({
  Sidebar: ({ onSelect }: { onSelect: (s: string, p?: string) => void }) => (
    <div data-testid="sidebar">
      <button data-testid="mock-session-select" onClick={() => onSelect('test-session')}>
        Select test-session
      </button>
    </div>
  ),
}))

// Mock MobileApp to prevent auth API calls
vi.mock('../mobile/MobileApp', () => ({
  default: () => <div data-testid="mobile-app">Mobile</div>,
}))

// Mock FloatingImperialStudy to prevent hook API calls
vi.mock('../shared/components/imperial-study/components/FloatingImperialStudy', () => ({
  FloatingImperialStudy: () => null,
}))

function renderApp(initialEntry = '/') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <App />
    </MemoryRouter>,
  )
}

describe('App', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('renders without crashing', () => {
    renderApp()
    expect(screen.getByTestId('sidebar')).toBeInTheDocument()
    expect(screen.getByTestId('terminal-tab')).toBeInTheDocument()
  })

  it('shows default ttyweb tab on mount', () => {
    renderApp()
    expect(screen.getByText('ttyweb')).toBeInTheDocument()
  })

  it('renders tab bar with toggle button', () => {
    renderApp()
    // Toggle button (◀ or ▶)
    const toggleBtn = screen.getByRole('button', { name: /◀|▶/ })
    expect(toggleBtn).toBeInTheDocument()
  })

  it('toggles sidebar on toggle button click', async () => {
    const user = userEvent.setup()
    renderApp()

    const sidebar = screen.getByTestId('sidebar')
    expect(sidebar).toBeInTheDocument()

    const toggleBtn = screen.getByRole('button', { name: /◀/ })
    await user.click(toggleBtn)

    expect(sidebar).not.toBeInTheDocument()
  })

  it('opens a new tab when session is selected from sidebar', async () => {
    const user = userEvent.setup()
    renderApp()

    const selectBtn = screen.getByTestId('mock-session-select')
    await user.click(selectBtn)

    expect(screen.getByText('test-session')).toBeInTheDocument()
    expect(screen.getByTestId('terminal-tab')).toHaveTextContent('test-session')
  })

  it('closes a tab via close button', async () => {
    renderApp()

    // Buttons: sidebar mock select, toggle, tab close, notepad
    const tabClose = screen.getByTitle('Close tab')
    expect(tabClose).toBeInTheDocument()

    fireEvent.click(tabClose)

    // After closing: sidebar mock select, toggle, notepad remain
    const buttonsAfter = screen.getAllByRole('button')
    expect(buttonsAfter.length).toBe(3)
    // Tab close button is gone
    expect(screen.queryByTitle('Close tab')).not.toBeInTheDocument()
  })

  it('renders MobileApp on /m route', () => {
    renderApp('/m')
    expect(screen.getByTestId('mobile-app')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar')).not.toBeInTheDocument()
  })
})
