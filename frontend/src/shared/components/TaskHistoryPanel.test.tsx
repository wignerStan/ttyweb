import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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
  {
    id: 2,
    conversation_id: 'c2',
    pane_key: 'main:0:0',
    user_message: 'Fix bug',
    assistant_message: null,
    conv_status: 'in_progress',
    started_at: Math.floor(Date.now() / 1000) - 60,
    completed_at: null,
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

  it('hides close button when embedded', () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} embedded />)
    expect(screen.queryByTitle('Close')).not.toBeInTheDocument()
  })

  it('renders pane context when paneKey is provided', async () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getAllByText('main:0').length).toBeGreaterThanOrEqual(1))
  })

  it('renders status badges for each conversation', async () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => {
      const badges = screen.getAllByText('completed')
      expect(badges.length).toBeGreaterThan(0)
    })
  })

  it('renders duration badge for each conversation', async () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => {
      // Duration is formatted as "Xm Ys" or "Xs"
      const durationBadges = screen.getAllByText(/\d+s|\d+m \d+s|\d+h \d+m/)
      expect(durationBadges.length).toBeGreaterThan(0)
    })
  })

  it('renders in_progress item with complete button', async () => {
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('Fix bug')).toBeInTheDocument())
    expect(screen.getByTitle('标记为已完成')).toBeInTheDocument()
  })

  it('markComplete calls API and updates state', async () => {
    const fetchMock = vi.fn()
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ conversations }),
      })
      .mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    const onStatusChange = vi.fn()
    renderWithProviders(
      <TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} onStatusChange={onStatusChange} />,
    )
    await waitFor(() => expect(screen.getByText('Fix bug')).toBeInTheDocument())
    await userEvent.click(screen.getByTitle('\u6807\u8BB0\u4E3A\u5DF2\u5B8C\u6210'))
    await waitFor(() => {
      const patchCalls = fetchMock.mock.calls.filter(
        (call: unknown[]) => (call[1] as { method?: string })?.method === 'PATCH',
      )
      expect(patchCalls.length).toBeGreaterThan(0)
    })
    expect(onStatusChange).toHaveBeenCalled()
  })

  it('applies task-history-embedded class when embedded', () => {
    const { container } = renderWithProviders(
      <TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} embedded />,
    )
    expect(container.querySelector('.task-history-embedded')).toBeInTheDocument()
  })

  it('applies aside element when not embedded', () => {
    const { container } = renderWithProviders(
      <TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />,
    )
    expect(container.querySelector('aside.task-history-panel')).toBeInTheDocument()
  })

  it('shows Untitled when user_message is empty', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            conversations: [
              {
                id: 10,
                conversation_id: 'c10',
                pane_key: 'main:0:0',
                user_message: '',
                assistant_message: null,
                conv_status: 'failed',
                started_at: Math.floor(Date.now() / 1000),
                completed_at: null,
              },
            ],
          }),
      }),
    )
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('Untitled')).toBeInTheDocument())
  })

  it('handles fetch failure gracefully', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('该 pane 暂无任务历史')).toBeInTheDocument())
  })

  it('markComplete refetches on API failure', async () => {
    let callCount = 0
    const fetchMock = vi.fn().mockImplementation(() => {
      callCount++
      if (callCount <= 1) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ conversations }),
        })
      }
      return Promise.reject(new Error('fail'))
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<TaskHistoryPanel paneKey="main:0:0" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('Fix bug')).toBeInTheDocument())
    await userEvent.click(screen.getByTitle('标记为已完成'))
    await waitFor(() => expect(callCount).toBeGreaterThanOrEqual(3))
  })
})
