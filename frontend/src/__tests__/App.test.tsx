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
      <button
        type="button"
        data-testid="mock-session-select"
        onClick={() => onSelect('test-session')}
      >
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

// Mock NotepadPanel
vi.mock('../notepad/NotepadPanel', () => ({
  NotepadPanel: () => <div data-testid="notepad-panel">Notepad</div>,
}))

// Mock KanbanBoard
vi.mock('../kanban', () => ({
  KanbanBoard: () => <div data-testid="kanban-board">Kanban</div>,
}))

// Mock ConversationList
vi.mock('../conversations/ConversationList', () => ({
  ConversationList: () => <div data-testid="conversation-list">ConvList</div>,
}))

// Mock ConversationViewer
vi.mock('../conversations/ConversationViewer', () => ({
  ConversationViewer: () => <div data-testid="conversation-viewer">ConvViewer</div>,
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

    // After closing: sidebar mock select, toggle, notepad, view tabs, kanban remain
    const buttonsAfter = screen.getAllByRole('button')
    expect(buttonsAfter.length).toBe(6)
    // Tab close button is gone
    expect(screen.queryByTitle('Close tab')).not.toBeInTheDocument()
  })

  it('renders MobileApp on /m route', () => {
    renderApp('/m')
    expect(screen.getByTestId('mobile-app')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar')).not.toBeInTheDocument()
  })

  it('shows Terminal and Conversations view tabs', () => {
    renderApp()
    expect(screen.getByText('Terminal')).toBeInTheDocument()
    expect(screen.getByText('Conversations')).toBeInTheDocument()
  })

  it('switches to Conversations view', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByText('Conversations'))
    expect(screen.getByTestId('conversation-list')).toBeInTheDocument()
    expect(screen.getByTestId('conversation-viewer')).toBeInTheDocument()
  })

  it('switches back to Terminal view from Conversations', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByText('Conversations'))
    await user.click(screen.getByText('Terminal'))
    expect(screen.getByTestId('terminal-tab')).toBeInTheDocument()
  })

  it('shows Kanban tab in tab bar', () => {
    renderApp()
    expect(screen.getByText('Kanban')).toBeInTheDocument()
  })

  it('switches to Kanban view', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByText('Kanban'))
    expect(screen.getByTestId('kanban-board')).toBeInTheDocument()
  })

  it('opens Notepad when notepad button clicked', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByTitle('Open Notepad'))
    expect(screen.getByTestId('notepad-panel')).toBeInTheDocument()
  })

  it('does not duplicate notepad tab on multiple clicks', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByTitle('Open Notepad'))
    await user.click(screen.getByTitle('Open Notepad'))
    const notepadTabs = screen.getAllByText('Notepad')
    // Only one tab label, not two
    expect(notepadTabs.length).toBeLessThanOrEqual(2) // one tab label + possibly the content
  })

  it('does not duplicate session tabs on multiple selections', async () => {
    const user = userEvent.setup()
    renderApp()
    await user.click(screen.getByTestId('mock-session-select'))
    await user.click(screen.getByTestId('mock-session-select'))
    // Only one test-session tab
    const tabs = screen.getAllByText('test-session')
    expect(tabs.length).toBe(1)
  })

  it('activates a tab when clicked', async () => {
    const user = userEvent.setup()
    renderApp()
    // Open a new tab
    await user.click(screen.getByTestId('mock-session-select'))
    // The ttyweb tab should still exist but not be active
    expect(screen.getByText('ttyweb')).toBeInTheDocument()
    expect(screen.getByText('test-session')).toBeInTheDocument()
  })

  it('shows notepad button in tab bar', () => {
    renderApp()
    expect(screen.getByTitle('Open Notepad')).toBeInTheDocument()
  })

  it('re-opens sidebar after toggle back', async () => {
    const user = userEvent.setup()
    renderApp()
    const toggleBtn = screen.getByRole('button', { name: /◀/ })
    await user.click(toggleBtn)
    expect(screen.queryByTestId('sidebar')).not.toBeInTheDocument()
    const expandBtn = screen.getByRole('button', { name: /▶/ })
    await user.click(expandBtn)
    expect(screen.getByTestId('sidebar')).toBeInTheDocument()
  })
})
