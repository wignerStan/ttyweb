import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { SessionGroup, TmuxSession } from '../../types'
import { GroupManager } from './GroupManager'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const groups: SessionGroup[] = [
  { id: 1, group_name: 'Work', sort_order: 0, session_count: 2 },
  { id: 2, group_name: 'Personal', sort_order: 1, session_count: 1 },
]

const sessions: TmuxSession[] = [{ sessionName: 'main', sessionId: '0', windows: [] }]

vi.stubGlobal(
  'fetch',
  vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ groups }) }),
)

describe('GroupManager snapshot', () => {
  it('renders group list with sessions', () => {
    const { container } = renderWithProviders(
      <GroupManager profileKey="default" sessions={sessions} onGroupsChanged={() => {}} />,
    )
    expect(container).toMatchSnapshot()
  })
})
