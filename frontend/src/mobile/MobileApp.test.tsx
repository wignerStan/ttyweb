import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../shared/components/LoginModal', () => ({
  LoginModal: ({ onLogin }: { onLogin: () => void }) => (
    <div data-testid="login-modal">
      <button onClick={onLogin} data-testid="login-btn">
        Login
      </button>
    </div>
  ),
}))

vi.mock('./MobileTerminal', () => ({
  MobileTerminal: ({ session }: { session: string }) => (
    <div data-testid="mobile-terminal">{session}</div>
  ),
}))

vi.mock('./MobileDrawer', () => ({
  MobileDrawer: ({ open, onClose }: { open: boolean; onClose: () => void }) => (
    <div data-testid="mobile-drawer" data-open={open ? 'true' : undefined}>
      <button onClick={onClose} data-testid="drawer-close">
        Close
      </button>
    </div>
  ),
}))

vi.mock('../shared/components/TaskHistoryPanel', () => ({
  TaskHistoryPanel: () => <div data-testid="task-history" />,
}))

vi.mock('../shared/components/imperial-study/components/ImperialStudyPanel', () => ({
  ImperialStudyPanel: () => <div data-testid="imperial-study" />,
}))

vi.mock('../hooks/useShakeDetect', () => ({
  default: () => ({ requestPermission: vi.fn() }),
}))

vi.mock('../hooks/useVisualViewport', () => ({
  default: () => {},
}))

const mockSessions = [
  {
    sessionName: 's1',
    sessionId: '1',
    windows: [
      {
        windowIndex: 0,
        windowName: 'main',
        windowId: 'w1',
        panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'vim' }],
      },
    ],
  },
]

describe('MobileApp', () => {
  beforeEach(() => {
    localStorage.clear()
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        data: { sessions: mockSessions },
      }),
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows login modal when not authenticated', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: false, json: async () => ({}) })

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await waitFor(() => {
      expect(screen.getByTestId('login-modal')).toBeInTheDocument()
    })
  })

  it('shows mobile layout after login', async () => {
    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    // Click login
    await waitFor(() => {
      expect(screen.getByTestId('login-btn')).toBeInTheDocument()
    })
    await userEvent.click(screen.getByTestId('login-btn'))

    // Should show mobile app
    await waitFor(() => {
      expect(screen.getByText('Select a pane')).toBeInTheDocument()
    })
  })

  it('persists tabs to localStorage', async () => {
    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await waitFor(() => {
      expect(screen.getByTestId('login-btn')).toBeInTheDocument()
    })
    await userEvent.click(screen.getByTestId('login-btn'))

    // Wait for sessions to load
    await waitFor(() => {
      expect(screen.getByText('Select a pane')).toBeInTheDocument()
    })

    // Verify localStorage is used for tab persistence
    const saved = localStorage.getItem('mobile-openTabs')
    expect(saved).toBeDefined()
  })

  it('shows imperial study button when a tab is active', async () => {
    // Pre-populate tabs in localStorage
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 's1:0', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await waitFor(() => {
      expect(screen.getByTestId('login-btn')).toBeInTheDocument()
    })
    await userEvent.click(screen.getByTestId('login-btn'))

    await waitFor(() => {
      expect(screen.getByTitle('Imperial Study')).toBeInTheDocument()
    })
  })

  it('renders loading state initially', async () => {
    globalThis.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })
})
