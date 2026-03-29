import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { TmuxSession } from '../types'
import { MobileDrawer } from './MobileDrawer'

vi.mock('../shared/components/TmuxTree', () => ({
  TmuxTree: ({
    sessions,
    onSelectPane,
  }: {
    sessions: TmuxSession[]
    onSelectPane: (paneId: string, paneName: string) => void
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
    render(
      <MobileDrawer
        open
        sessions={mockSessions}
        currentProfile={null}
        groups={[]}
        onSelectPane={onSelectPane}
        onClose={onClose}
        onRefresh={onRefresh}
        onLogout={onLogout}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
      />,
    )

    const drawer = screen.getByTestId('tmux-tree').closest('.mobile-drawer')
    expect(drawer?.className).toContain('open')
  })

  it('does not apply open class when closed', () => {
    render(
      <MobileDrawer
        open={false}
        sessions={mockSessions}
        currentProfile={null}
        groups={[]}
        onSelectPane={onSelectPane}
        onClose={onClose}
        onRefresh={onRefresh}
        onLogout={onLogout}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
      />,
    )

    const drawer = screen.getByTestId('tmux-tree').closest('.mobile-drawer')
    expect(drawer?.className).not.toContain('open')
  })

  it('renders session list from TmuxTree', () => {
    render(
      <MobileDrawer
        open
        sessions={mockSessions}
        currentProfile={null}
        groups={[]}
        onSelectPane={onSelectPane}
        onClose={onClose}
        onRefresh={onRefresh}
        onLogout={onLogout}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
      />,
    )

    expect(screen.getByTestId('pane-sess-1')).toBeInTheDocument()
    expect(screen.getByTestId('pane-sess-2')).toBeInTheDocument()
  })

  it('calls onSelectPane and onClose when pane is selected', async () => {
    const user = userEvent.setup()
    render(
      <MobileDrawer
        open
        sessions={mockSessions}
        currentProfile={null}
        groups={[]}
        onSelectPane={onSelectPane}
        onClose={onClose}
        onRefresh={onRefresh}
        onLogout={onLogout}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
      />,
    )

    await user.click(screen.getByTestId('pane-sess-1'))

    expect(onSelectPane).toHaveBeenCalledWith('%0', 'sess-1:0')
    expect(onClose).toHaveBeenCalled()
  })

  it('calls onRefresh when refresh button clicked', async () => {
    const user = userEvent.setup()
    render(
      <MobileDrawer
        open
        sessions={mockSessions}
        currentProfile={null}
        groups={[]}
        onSelectPane={onSelectPane}
        onClose={onClose}
        onRefresh={onRefresh}
        onLogout={onLogout}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
      />,
    )

    await user.click(screen.getByTitle('Refresh'))
    expect(onRefresh).toHaveBeenCalled()
  })

  it('calls onLogout when logout button clicked', async () => {
    const user = userEvent.setup()
    render(
      <MobileDrawer
        open
        sessions={mockSessions}
        currentProfile={null}
        groups={[]}
        onSelectPane={onSelectPane}
        onClose={onClose}
        onRefresh={onRefresh}
        onLogout={onLogout}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
      />,
    )

    await user.click(screen.getByTitle('Sign out'))
    expect(onLogout).toHaveBeenCalled()
  })
})
