import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { SessionGroup, TmuxSession } from '../../types'
import { GroupManager } from './GroupManager'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
}))

const groups: SessionGroup[] = [
  { id: 1, group_name: 'Work', sort_order: 0, session_count: 2 },
  { id: 2, group_name: 'Personal', sort_order: 1, session_count: 1 },
]

const sessions: TmuxSession[] = [{ sessionName: 'main', sessionId: '0', windows: [] }]

const noop = () => {}

function mockFetchJSON(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(data) })
}

describe('GroupManager', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetchJSON({ groups }))
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders group list', async () => {
    renderWithProviders(
      <GroupManager profileKey="default" sessions={sessions} onGroupsChanged={noop} />,
    )
    await waitFor(() => expect(screen.getByText('Work')).toBeInTheDocument())
    expect(screen.getByText('Personal')).toBeInTheDocument()
  })

  it('shows empty state when no groups', async () => {
    vi.stubGlobal('fetch', mockFetchJSON({ groups: [] }))
    renderWithProviders(<GroupManager profileKey="default" sessions={[]} onGroupsChanged={noop} />)
    await waitFor(() => expect(screen.getByText('No groups yet')).toBeInTheDocument())
  })

  it('creates a new group', async () => {
    const fetchMock = mockFetchJSON({ groups })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ groups }) })
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ id: 3, group_name: 'Test' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const onChanged = vi.fn()
    renderWithProviders(
      <GroupManager profileKey="default" sessions={sessions} onGroupsChanged={onChanged} />,
    )
    await waitFor(() => expect(screen.getByText('Work')).toBeInTheDocument())
    await userEvent.click(screen.getByTitle('Create group'))
    await userEvent.type(screen.getByPlaceholderText('Group name...'), 'Test')
    await userEvent.click(
      screen.getByPlaceholderText('Group name...').nextElementSibling as HTMLElement,
    )
    await waitFor(() => expect(onChanged).toHaveBeenCalled())
  })

  it('edits a group name', async () => {
    const fetchMock = mockFetchJSON({ groups })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ groups }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(
      <GroupManager profileKey="default" sessions={sessions} onGroupsChanged={noop} />,
    )
    await waitFor(() => expect(screen.getByText('Work')).toBeInTheDocument())
    const renameBtns = screen.getAllByTitle('Rename')
    expect(renameBtns[0]).toBeTruthy()
    await userEvent.click(renameBtns[0]!)
    const input = screen.getByDisplayValue('Work')
    await userEvent.clear(input)
    await userEvent.type(input, 'Renamed')
    await userEvent.click(input.nextElementSibling as HTMLElement)
    await waitFor(() => expect(screen.getByText('Renamed')).toBeInTheDocument())
  })

  it('deletes a group with confirmation', async () => {
    const fetchMock = mockFetchJSON({ groups })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ groups }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(
      <GroupManager profileKey="default" sessions={sessions} onGroupsChanged={noop} />,
    )
    await waitFor(() => expect(screen.getByText('Work')).toBeInTheDocument())
    const deleteBtns = screen.getAllByTitle('Delete')
    expect(deleteBtns[0]).toBeTruthy()
    await userEvent.click(deleteBtns[0]!)
    await waitFor(() => expect(screen.queryByText('Work')).not.toBeInTheDocument())
  })
})
