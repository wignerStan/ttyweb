import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { Profile, TmuxSession } from '../types'
import { MobileDrawer } from './MobileDrawer'

vi.mock('../shared/components/TmuxTree', () => ({
  TmuxTree: ({ sessions }: { sessions: TmuxSession[]; [key: string]: unknown }) => (
    <div data-testid="tmux-tree">
      {sessions.map((s) => (
        <span key={s.sessionName}>{s.sessionName}</span>
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
    sessionName: 'main',
    sessionId: '1',
    windows: [
      {
        windowIndex: 0,
        windowName: 'editor',
        windowId: 'w1',
        panes: [{ paneId: '%0', paneTitle: 'bash', paneCommand: 'vim' }],
      },
    ],
  },
  {
    sessionName: 'work',
    sessionId: '2',
    windows: [
      {
        windowIndex: 0,
        windowName: 'terminal',
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

describe('MobileDrawer snapshot', () => {
  it('renders drawer panel open with sessions and profile', () => {
    const { container } = render(
      <MobileDrawer
        open={true}
        sessions={mockSessions}
        currentProfile={mockProfile}
        groups={[]}
        statusRefreshToken={0}
        onProfileChange={vi.fn()}
        onGroupsChanged={vi.fn()}
        onSelectPane={vi.fn()}
        onClose={vi.fn()}
        onRefresh={vi.fn()}
        onLogout={vi.fn()}
      />,
    )
    expect(container).toMatchSnapshot()
  })
})
