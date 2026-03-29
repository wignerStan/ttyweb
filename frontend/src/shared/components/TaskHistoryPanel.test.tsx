import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { TaskHistoryPanel } from './TaskHistoryPanel'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const conversations = [
  {
    id: 1,
    conversation_id: 'c1',
    pane_key: 'main:0:0',
    user_message: 'Build the project',
    assistant_message: 'Done!',
    conv_status: 'completed',
    started_at: Math.floor(Date.now() / 1000) - 120,
    completed_at: Math.floor(Date.now() / 1000),
  },
]

describe('TaskHistoryPanel', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ conversations }) }),
    )
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders event list', async () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('Build the project')).toBeInTheDocument())
    expect(screen.getByText('Done!')).toBeInTheDocument()
  })

  it('shows loading state', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => new Promise(() => {})),
    )
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows empty state when no pane key', () => {
    renderWithProviders(<TaskHistoryPanel paneKey={null} onClose={() => {}} />)
    expect(screen.getByText('点击 pane 的状态图标查看任务历史')).toBeInTheDocument()
  })

  it('shows empty state when no conversations', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ conversations: [] }) }),
    )
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('该 pane 暂无任务历史')).toBeInTheDocument())
  })

  it('shows close button when not embedded', () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    expect(screen.getByTitle('Close')).toBeInTheDocument()
  })

  it('refreshes on button click', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ conversations }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('Build the project')).toBeInTheDocument())
    await userEvent.click(screen.getByTitle('Refresh'))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  })
})
