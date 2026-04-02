import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { TmuxSession } from '../../types'
import { NewTmuxButton } from './NewTmuxButton'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
}))

const sessions: TmuxSession[] = [
  { sessionName: 'main', sessionId: '0', windows: [] },
  { sessionName: 'dev', sessionId: '1', windows: [] },
]

function mockFetchJSON(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(data) })
}

describe('NewTmuxButton', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      mockFetchJSON({ dirs: [{ name: 'projects', path: '/home/u/projects' }] }),
    )
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('opens and closes menu', async () => {
    renderWithProviders(<NewTmuxButton sessions={sessions} />)
    const btn = screen.getByTitle('New session / window')
    await userEvent.click(btn)
    await waitFor(() => expect(screen.getByText('New Session')).toBeInTheDocument())
    expect(screen.getByText('New Window')).toBeInTheDocument()
    await userEvent.click(btn)
    expect(screen.queryByText('New Session')).not.toBeInTheDocument()
  })

  it('renders session name input', async () => {
    renderWithProviders(<NewTmuxButton sessions={sessions} />)
    await userEvent.click(screen.getByTitle('New session / window'))
    await waitFor(() =>
      expect(screen.getByPlaceholderText('Session name (optional)')).toBeInTheDocument(),
    )
  })

  it('renders dir chips', async () => {
    renderWithProviders(<NewTmuxButton sessions={sessions} />)
    await userEvent.click(screen.getByTitle('New session / window'))
    await waitFor(() => {
      const chips = screen.getAllByText('projects')
      expect(chips.length).toBeGreaterThanOrEqual(1)
    })
    const tildeChips = screen.getAllByText('~')
    expect(tildeChips.length).toBeGreaterThanOrEqual(1)
  })

  it('creates session via POST', async () => {
    const fetchMock = mockFetchJSON({ dirs: [] })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ dirs: [] }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    const onCreated = vi.fn()
    renderWithProviders(<NewTmuxButton sessions={sessions} onCreated={onCreated} />)
    await userEvent.click(screen.getByTitle('New session / window'))
    await waitFor(() => expect(screen.getByText('Create Session')).toBeInTheDocument())
    await userEvent.type(screen.getByPlaceholderText('Session name (optional)'), 'mysess')
    await userEvent.click(screen.getByText('Create Session'))
    await waitFor(() => expect(onCreated).toHaveBeenCalled())
  })

  it('creates window via POST', async () => {
    const fetchMock = mockFetchJSON({ dirs: [] })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ dirs: [] }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    const onCreated = vi.fn()
    renderWithProviders(<NewTmuxButton sessions={sessions} onCreated={onCreated} />)
    await userEvent.click(screen.getByTitle('New session / window'))
    await waitFor(() => expect(screen.getByText('Add Window')).toBeInTheDocument())
    await userEvent.click(screen.getByText('Add Window'))
    await waitFor(() => expect(onCreated).toHaveBeenCalled())
  })

  it('closes menu on outside click', async () => {
    renderWithProviders(
      <div>
        <div data-testid="outside">outside</div>
        <NewTmuxButton sessions={sessions} />
      </div>,
    )
    await userEvent.click(screen.getByTitle('New session / window'))
    await waitFor(() => expect(screen.getByText('New Session')).toBeInTheDocument())
    await userEvent.click(screen.getByTestId('outside'))
    expect(screen.queryByText('New Session')).not.toBeInTheDocument()
  })
})
