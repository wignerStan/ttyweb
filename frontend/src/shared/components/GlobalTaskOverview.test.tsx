import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { mockFetchHttpError } from '../../test-helpers'
import { renderWithProviders } from '../../test-utils'
import { GlobalTaskOverview } from './GlobalTaskOverview'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
}))

const tasks = [
  {
    id: 1,
    task_title: 'Build project',
    task_status: 'in_progress',
    started_at: Math.floor(Date.now() / 1000) - 300,
    completed_at: 0,
    paneKey: 'main:0:0',
    session_name: 'main',
    window_index: 0,
    pane_index: '%0',
    mtime: Math.floor(Date.now() / 1000),
  },
]

describe('GlobalTaskOverview', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ tasks }) }),
    )
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders task overview with tasks', async () => {
    renderWithProviders(<GlobalTaskOverview onSelectPane={() => {}} />)
    await waitFor(() => expect(screen.getByText('Build project')).toBeInTheDocument())
    expect(screen.getByText('In Progress')).toBeInTheDocument()
  })

  it('shows loading state', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => new Promise(() => {})),
    )
    renderWithProviders(<GlobalTaskOverview onSelectPane={() => {}} />)
    expect(screen.getByText('Loading tasks...')).toBeInTheDocument()
  })

  it('shows error state', async () => {
    vi.stubGlobal('fetch', mockFetchHttpError(500))
    renderWithProviders(<GlobalTaskOverview onSelectPane={() => {}} />)
    await waitFor(() => expect(screen.getByText('Failed to fetch tasks')).toBeInTheDocument())
    expect(screen.getByText('Retry')).toBeInTheDocument()
  })

  it('shows empty state when no tasks', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ tasks: [] }) }),
    )
    renderWithProviders(<GlobalTaskOverview onSelectPane={() => {}} />)
    await waitFor(() =>
      expect(screen.getByText('No tasks found across sessions')).toBeInTheDocument(),
    )
  })

  it('calls onSelectPane when task is clicked', async () => {
    const onSelect = vi.fn()
    renderWithProviders(<GlobalTaskOverview onSelectPane={onSelect} />)
    await waitFor(() => expect(screen.getByText('Build project')).toBeInTheDocument())
    await userEvent.click(screen.getByText('Build project'))
    expect(onSelect).toHaveBeenCalled()
  })

  it('retries on retry button click', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, status: 500, json: () => Promise.resolve({}) })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ tasks: [] }) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<GlobalTaskOverview onSelectPane={() => {}} />)
    await waitFor(() => expect(screen.getByText('Retry')).toBeInTheDocument())
    await userEvent.click(screen.getByText('Retry'))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })
})
