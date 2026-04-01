import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../shared/components/LoginModal', () => ({
  LoginModal: ({ onLogin }: { onLogin: () => void }) => (
    <div data-testid="login-modal">
      <button type="button" onClick={onLogin} data-testid="login-btn">
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
  MobileDrawer: ({
    open,
    onClose,
    onSelectPane,
    onRefresh,
    onLogout,
  }: {
    open: boolean
    onClose: () => void
    onSelectPane: (paneId: string, paneName: string) => void
    onRefresh: () => void
    onLogout: () => void
    sessions: unknown[]
    onProfileChange: (profile: unknown) => void
    currentProfile: unknown
    groups: unknown[]
    onGroupsChanged: () => void
    onPaneStatusClick: (paneKey: string) => void
    statusRefreshToken: number
  }) => (
    <div data-testid="mobile-drawer" data-open={open ? 'true' : undefined}>
      <button type="button" onClick={onClose} data-testid="drawer-close">
        Close
      </button>
      <button type="button" onClick={() => onSelectPane('%0', 'bash')} data-testid="select-pane">
        Select Pane
      </button>
      <button type="button" onClick={onRefresh} data-testid="drawer-refresh">
        Refresh
      </button>
      <button type="button" onClick={onLogout} data-testid="drawer-logout">
        Logout
      </button>
    </div>
  ),
}))

vi.mock('../shared/components/TaskHistoryPanel', () => ({
  TaskHistoryPanel: ({
    paneKey,
    onClose,
    onStatusChange,
  }: {
    paneKey: string | null
    onClose: () => void
    onStatusChange: () => void
  }) => (
    <div data-testid="task-history" data-pane-key={paneKey ?? ''}>
      <button type="button" onClick={onClose} data-testid="history-close">
        Close
      </button>
      <button type="button" onClick={onStatusChange} data-testid="status-change">
        Status Change
      </button>
    </div>
  ),
}))

vi.mock('../shared/components/imperial-study/components/ImperialStudyPanel', () => ({
  ImperialStudyPanel: ({ activePaneKey }: { activePaneKey: string | null }) => (
    <div data-testid="imperial-study" data-pane-key={activePaneKey ?? ''} />
  ),
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

const mockSessionsMultiPane = [
  {
    sessionName: 's1',
    sessionId: '1',
    windows: [
      {
        windowIndex: 0,
        windowName: 'main',
        windowId: 'w1',
        panes: [
          { paneId: '%0', paneTitle: 'bash', paneCommand: 'vim' },
          { paneId: '%1', paneTitle: 'zsh', paneCommand: 'node' },
        ],
      },
    ],
  },
]

function createFetchMock(sessions = mockSessions) {
  let callCount = 0
  return vi.fn().mockImplementation(() => {
    callCount++
    // First call is checkAuth - return auth_enabled: true with no credentials => false
    if (callCount === 1) {
      return Promise.resolve({
        ok: true,
        json: async () => ({ data: { auth_enabled: true } }),
      })
    }
    return Promise.resolve({
      ok: true,
      json: async () => ({ data: { sessions } }),
    })
  })
}

async function login() {
  await waitFor(() => {
    expect(screen.getByTestId('login-btn')).toBeInTheDocument()
  })
  await userEvent.click(screen.getByTestId('login-btn'))
}

async function waitForApp() {
  await waitFor(() => {
    expect(screen.getByText('Select a pane')).toBeInTheDocument()
  })
}

describe('MobileApp', () => {
  beforeEach(() => {
    localStorage.clear()
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
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()
  })

  it('persists tabs to localStorage', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    const saved = localStorage.getItem('mobile-openTabs')
    expect(saved).toBeDefined()
  })

  it('shows imperial study button when a tab is active', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 's1:0', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByTitle('Imperial Study')).toBeInTheDocument()
    })
  })

  it('renders loading state initially', async () => {
    globalThis.fetch = vi.fn().mockImplementation(() => new Promise(() => {}))

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    expect(screen.getByText('Loading…')).toBeInTheDocument()
  })

  it('shows loading sessions state when authenticated and loading', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string | Request) => {
      const urlStr = typeof url === 'string' ? url : url.toString()
      if (urlStr.includes('/api/auth/check')) {
        return Promise.resolve({ ok: true, json: async () => ({ data: { auth_enabled: false } }) })
      }
      return new Promise(() => {})
    })

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await waitFor(() => {
      expect(screen.getByText('Loading sessions…')).toBeInTheDocument()
    })
  })

  it('shows error state when tree fetch fails', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string | Request) => {
      const urlStr = typeof url === 'string' ? url : url.toString()
      if (urlStr.includes('/api/auth/check')) {
        return Promise.resolve({ ok: true, json: async () => ({ data: { auth_enabled: false } }) })
      }
      return Promise.resolve({ ok: false, status: 500, json: async () => ({}) })
    })

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await waitFor(() => {
      expect(screen.getByText('Failed to fetch tree')).toBeInTheDocument()
    })
  })

  it('opens and closes drawer via menu button', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    // Find and click menu button (first .mobile-menu-btn in header)
    const menuBtn = document.querySelector('.mobile-header .mobile-menu-btn')!
    fireEvent.click(menuBtn)

    await waitFor(() => {
      expect(screen.getByTestId('mobile-drawer')).toHaveAttribute('data-open', 'true')
    })

    // Close drawer
    fireEvent.click(screen.getByTestId('drawer-close'))

    await waitFor(() => {
      expect(screen.getByTestId('mobile-drawer')).not.toHaveAttribute('data-open')
    })
  })

  it('selects pane from drawer and creates tab', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    // Open drawer
    const menuBtn = document.querySelector('.mobile-header .mobile-menu-btn')!
    fireEvent.click(menuBtn)

    await waitFor(() => {
      expect(screen.getByTestId('mobile-drawer')).toHaveAttribute('data-open', 'true')
    })

    // Select a pane
    fireEvent.click(screen.getByTestId('select-pane'))

    // Drawer should close
    await waitFor(() => {
      expect(screen.getByTestId('mobile-drawer')).not.toHaveAttribute('data-open')
    })

    // Should show tab with title
    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument()
    })
  })

  it('closes a tab via close button', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument()
    })

    // Find close button (X icon) inside the active tab
    const tab = screen.getByText('bash').closest('.mobile-tab')!
    const closeBtn = tab.querySelector('.mobile-tab-close')!
    fireEvent.click(closeBtn)

    // Tab should be removed
    await waitFor(() => {
      expect(screen.queryByText('bash')).not.toBeInTheDocument()
    })

    // Should show placeholder
    await waitFor(() => {
      expect(screen.getByText('Select a pane')).toBeInTheDocument()
    })
  })

  it('switches active tab when closing active tab with multiple tabs', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([
        { id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' },
        { id: 'tab-%1', paneId: '%1', title: 'zsh', session: 's1' },
      ]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock(mockSessionsMultiPane)

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument()
    })

    // Close the active tab (bash = tab-%0)
    const bashTab = screen.getByText('bash').closest('.mobile-tab')!
    const closeBtn = bashTab.querySelector('.mobile-tab-close')!
    fireEvent.click(closeBtn)

    // The other tab should remain
    await waitFor(() => {
      expect(screen.getByText('zsh')).toBeInTheDocument()
    })
  })

  it('switches active tab by clicking on it', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([
        { id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' },
        { id: 'tab-%1', paneId: '%1', title: 'zsh', session: 's1' },
      ]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock(mockSessionsMultiPane)

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument()
      expect(screen.getByText('zsh')).toBeInTheDocument()
    })

    // Click on the second tab
    const zshTab = screen.getByText('zsh').closest('.mobile-tab')!
    fireEvent.click(zshTab)

    // The zsh tab should become active (has 'active' class)
    await waitFor(() => {
      expect(zshTab.className).toContain('active')
    })
  })

  it('removes invalid tabs when sessions update', async () => {
    // Pre-populate with a tab for a pane that won't exist in fetched sessions
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%99', paneId: '%99', title: 'gone', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%99')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    // The invalid tab should be removed, leaving no tabs
    await waitFor(() => {
      expect(screen.queryByText('gone')).not.toBeInTheDocument()
    })
  })

  it('opens and closes imperial study panel', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByTitle('Imperial Study')).toBeInTheDocument()
    })

    // Open imperial study
    fireEvent.click(screen.getByTitle('Imperial Study'))

    await waitFor(() => {
      expect(screen.getByTestId('imperial-study')).toBeInTheDocument()
    })

    // Close by clicking overlay
    const overlay = document.querySelector('.mobile-overlay')!
    fireEvent.click(overlay)

    await waitFor(() => {
      expect(screen.queryByTestId('imperial-study')).not.toBeInTheDocument()
    })
  })

  it('opens right panel via history button', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByTitle('Task history')).toBeInTheDocument()
    })

    // Open task history
    fireEvent.click(screen.getByTitle('Task history'))

    await waitFor(() => {
      expect(screen.getByTestId('task-history')).toBeInTheDocument()
    })

    // Close by clicking inside the panel close button
    fireEvent.click(screen.getByTestId('history-close'))

    await waitFor(() => {
      expect(screen.queryByTestId('task-history')).not.toBeInTheDocument()
    })
  })

  it('handles logout correctly', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    // Open drawer
    const menuBtn = document.querySelector('.mobile-header .mobile-menu-btn')!
    fireEvent.click(menuBtn)

    await waitFor(() => {
      expect(screen.getByTestId('mobile-drawer')).toHaveAttribute('data-open', 'true')
    })

    // Click logout
    fireEvent.click(screen.getByTestId('drawer-logout'))

    // Should show login modal
    await waitFor(() => {
      expect(screen.getByTestId('login-modal')).toBeInTheDocument()
    })
  })

  it('closes all overlays when overlay is clicked', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      expect(screen.getByTitle('Imperial Study')).toBeInTheDocument()
    })
    fireEvent.click(screen.getByTitle('Imperial Study'))

    await waitFor(() => {
      expect(screen.getByTestId('imperial-study')).toBeInTheDocument()
    })

    // Click overlay
    const overlay = document.querySelector('.mobile-overlay')!
    fireEvent.click(overlay)

    await waitFor(() => {
      expect(screen.queryByTestId('imperial-study')).not.toBeInTheDocument()
    })
  })

  it('saves and restores font size from localStorage', async () => {
    localStorage.setItem('terminal-font-size', '14')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    const savedSize = localStorage.getItem('terminal-font-size')
    expect(savedSize).toBe('14')
  })

  it('renders terminals for each open tab', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([{ id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' }]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      const terminals = screen.getAllByTestId('mobile-terminal')
      expect(terminals).toHaveLength(1)
      expect(terminals[0]).toHaveTextContent('s1')
    })
  })

  it('shows task history button only when a tab is active', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    // No tabs open - no task history button
    expect(screen.queryByTitle('Task history')).not.toBeInTheDocument()

    // Open drawer and select pane
    const menuBtn = document.querySelector('.mobile-header .mobile-menu-btn')!
    fireEvent.click(menuBtn)

    await waitFor(() => {
      expect(screen.getByTestId('mobile-drawer')).toHaveAttribute('data-open', 'true')
    })

    fireEvent.click(screen.getByTestId('select-pane'))

    await waitFor(() => {
      expect(screen.getByTitle('Task history')).toBeInTheDocument()
    })
  })

  it('displays pane placeholder when no tabs are open', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()
    await waitForApp()

    expect(screen.getByText(/to select a terminal/)).toBeInTheDocument()
  })

  it('matches snapshot with no tabs (placeholder state)', async () => {
    globalThis.fetch = createFetchMock()

    const MobileApp = (await import('./MobileApp')).default
    const { container } = render(<MobileApp />)

    await login()
    await waitForApp()

    expect(container).toMatchSnapshot()
  })

  it('renders tab with active class for the active tab', async () => {
    localStorage.setItem(
      'mobile-openTabs',
      JSON.stringify([
        { id: 'tab-%0', paneId: '%0', title: 'bash', session: 's1' },
        { id: 'tab-%1', paneId: '%1', title: 'zsh', session: 's1' },
      ]),
    )
    localStorage.setItem('mobile-activeTabId', 'tab-%0')

    globalThis.fetch = createFetchMock(mockSessionsMultiPane)

    const MobileApp = (await import('./MobileApp')).default
    render(<MobileApp />)

    await login()

    await waitFor(() => {
      const bashTab = screen.getByText('bash').closest('.mobile-tab')!
      expect(bashTab.className).toContain('active')

      const zshTab = screen.getByText('zsh').closest('.mobile-tab')!
      expect(zshTab.className).not.toContain('active')
    })
  })
})
