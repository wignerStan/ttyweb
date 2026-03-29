import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Profile, TmuxSession } from '../types'
import { MobileDrawer } from './MobileDrawer'

vi.mock('../shared/components/TmuxTree', () => ({
  TmuxTree: ({
    sessions,
    onSelectPane,
    onPaneStatusClick,
  }: {
    sessions: TmuxSession[]
    onSelectPane: (paneId: string, paneName: string) => void
    onPaneStatusClick?: (paneKey: string) => void
    [key: string]: unknown
  }) => (
    <div data-testid="tmux-tree">
      {sessions.map((s) => (
        <button
          key={s.sessionName}
          onClick={() => onSelectPane('%0', `${s.sessionName}:0`)}
          data-testid={`pane-${s.sessionName}`}
          type="button"
        >
          {s.sessionName}
        </button>
      ))}
      <button
        data-testid="pane-status-click"
        onClick={() => onPaneStatusClick?.('main:0:0')}
        type="button"
      >
        Status Click
      </button>
    </div>
  ),
}))

vi.mock('../shared/components/ProfileSelector', () => ({
  ProfileSelector: () => <div data-testid="profile-selector" />,
}))

vi.mock('../shared/components/TaskStatBadges', () => ({
  TaskStatBadges: () => <div data-testid="task-stat-badges" />,
}))

vi.mock('../shared/components/GroupManager', () => ({
  GroupManager: () => <div data-testid="group-manager" />,
}))

const mockSessions: TmuxSession[] = [
  {
    sessionName: 'sess-1',
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
  {
    sessionName: 'sess-2',
    sessionId: '2',
    windows: [
      {
        windowIndex: 0,
        windowName: 'main',
        windowId: 'w2',
        panes: [{ paneId: '%1', paneTitle: 'bash', paneCommand: 'node' }],
      },
    ],
  },
]

const mockProfile: Profile = {
  id: 1,
  profile_key: 'default',
  name: 'Default',
  sort_order: 0,
}

function renderDrawer(overrides: Record<string, unknown> = {}) {
  const props = {
    open: true,
    sessions: mockSessions,
    currentProfile: null as Profile | null,
    groups: [],
    onSelectPane: vi.fn<(paneId: string, paneName: string) => void>(),
    onClose: vi.fn<() => void>(),
    onRefresh: vi.fn<() => void>(),
    onLogout: vi.fn<() => void>(),
    onProfileChange: vi.fn(),
    onGroupsChanged: vi.fn(),
    ...overrides,
  }
  return {
    ...render(<MobileDrawer {...props} />),
    props,
  }
}

describe('MobileDrawer', () => {
  let onSelectPane: ReturnType<typeof vi.fn<(paneId: string, paneName: string) => void>>
  let onClose: ReturnType<typeof vi.fn<() => void>>
  let onRefresh: ReturnType<typeof vi.fn<() => void>>
  let onLogout: ReturnType<typeof vi.fn<() => void>>

  beforeEach(() => {
    onSelectPane = vi.fn<(paneId: string, paneName: string) => void>()
    onClose = vi.fn<() => void>()
    onRefresh = vi.fn<() => void>()
    onLogout = vi.fn<() => void>()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('applies open class when open prop is true', () => {
    renderDrawer()
    const drawer = screen.getByTestId('tmux-tree').closest('.mobile-drawer')
    expect(drawer?.className).toContain('open')
  })

  it('does not apply open class when closed', () => {
    renderDrawer({ open: false })
    const drawer = screen.getByTestId('tmux-tree').closest('.mobile-drawer')
    expect(drawer?.className).not.toContain('open')
  })

  it('renders session list from TmuxTree', () => {
    renderDrawer()
    expect(screen.getByTestId('pane-sess-1')).toBeInTheDocument()
    expect(screen.getByTestId('pane-sess-2')).toBeInTheDocument()
  })

  it('calls onSelectPane and onClose when pane is selected', async () => {
    const user = userEvent.setup()
    const { props } = renderDrawer({ onSelectPane, onClose })
    await user.click(screen.getByTestId('pane-sess-1'))
    expect(props.onSelectPane).toHaveBeenCalledWith('%0', 'sess-1:0')
    expect(props.onClose).toHaveBeenCalled()
  })

  it('calls onRefresh when refresh button clicked', async () => {
    const user = userEvent.setup()
    const { props } = renderDrawer({ onRefresh })
    await user.click(screen.getByTitle('Refresh'))
    expect(props.onRefresh).toHaveBeenCalled()
  })

  it('calls onLogout when logout button clicked', async () => {
    const user = userEvent.setup()
    const { props } = renderDrawer({ onLogout })
    await user.click(screen.getByTitle('Sign out'))
    expect(props.onLogout).toHaveBeenCalled()
  })

  it('calls onClose when close button clicked', async () => {
    const user = userEvent.setup()
    const { props } = renderDrawer({ onClose })
    await user.click(screen.getByTitle('Close'))
    expect(props.onClose).toHaveBeenCalled()
  })

  it('renders ProfileSelector', () => {
    renderDrawer()
    expect(screen.getByTestId('profile-selector')).toBeInTheDocument()
  })

  it('renders TaskStatBadges', () => {
    renderDrawer()
    expect(screen.getByTestId('task-stat-badges')).toBeInTheDocument()
  })

  it('shows GroupManager when settings clicked and profile is set', async () => {
    const user = userEvent.setup()
    renderDrawer({ currentProfile: mockProfile })
    await user.click(screen.getByTitle('Manage groups'))
    expect(screen.getByTestId('group-manager')).toBeInTheDocument()
  })

  it('hides GroupManager when profile is null', async () => {
    const user = userEvent.setup()
    renderDrawer({ currentProfile: null })
    await user.click(screen.getByTitle('Manage groups'))
    expect(screen.queryByTestId('group-manager')).not.toBeInTheDocument()
  })

  it('toggles GroupManager visibility on settings click', async () => {
    const user = userEvent.setup()
    renderDrawer({ currentProfile: mockProfile })
    await user.click(screen.getByTitle('Manage groups'))
    expect(screen.getByTestId('group-manager')).toBeInTheDocument()
    await user.click(screen.getByTitle('Manage groups'))
    expect(screen.queryByTestId('group-manager')).not.toBeInTheDocument()
  })

  it('calls onPaneStatusClick and closes drawer', async () => {
    const user = userEvent.setup()
    const onPaneStatusClick = vi.fn<(paneKey: string) => void>()
    const { props } = renderDrawer({ onPaneStatusClick, onClose })
    await user.click(screen.getByTestId('pane-status-click'))
    expect(onPaneStatusClick).toHaveBeenCalledWith('main:0:0')
    expect(props.onClose).toHaveBeenCalled()
  })

  it('renders as aside element with mobile-drawer class', () => {
    const { container } = renderDrawer()
    expect(container.querySelector('aside.mobile-drawer')).toBeInTheDocument()
  })
})
