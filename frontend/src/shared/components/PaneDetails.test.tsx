import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { AiConversation } from '../../types'
import { PaneDetails } from './PaneDetails'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
}))
vi.mock('../../hooks/useAIConversations', () => ({
  useAIConversations: () => ({
    conversations: mockConversations,
    loading: false,
    refetch: vi.fn(),
  }),
}))

const mockConversations: AiConversation[] = [
  {
    conversation_id: 'conv-1',
    pane_key: 'main:0:0',
    user_message: 'Fix the bug',
    assistant_message: 'I fixed it',
    conv_status: 'completed',
    started_at: Math.floor(Date.now() / 1000) - 60,
    completed_at: Math.floor(Date.now() / 1000),
  },
]

describe('PaneDetails', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue({ ok: true, json: () => Promise.resolve({ tasks: [], panes: [] }) }),
    )
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders nothing when paneKey is null', () => {
    const { container } = renderWithProviders(
      <PaneDetails paneKey={null} profileKey="default" onClose={() => {}} />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders pane info header', () => {
    renderWithProviders(<PaneDetails paneKey="main:0:0" profileKey="default" onClose={() => {}} />)
    expect(screen.getByText('Execution History')).toBeInTheDocument()
  })

  it('shows AI conversations', () => {
    renderWithProviders(<PaneDetails paneKey="main:0:0" profileKey="default" onClose={() => {}} />)
    expect(screen.getByText('Fix the bug')).toBeInTheDocument()
  })

  it('renders close button and calls onClose', async () => {
    const onClose = vi.fn()
    renderWithProviders(<PaneDetails paneKey="main:0:0" profileKey="default" onClose={onClose} />)
    const closeBtn = document.querySelector('.drawer-close') as HTMLElement
    expect(closeBtn).toBeInTheDocument()
    await userEvent.click(closeBtn)
    expect(onClose).toHaveBeenCalled()
  })

  it('shows task list when tasks exist', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            tasks: [
              {
                id: 1,
                task_title: 'Manual task',
                task_status: 'in_progress',
                started_at: Math.floor(Date.now() / 1000),
                completed_at: 0,
                paneKey: 'main:0:0',
              },
            ],
          }),
      }),
    )
    renderWithProviders(<PaneDetails paneKey="main:0:0" profileKey="default" onClose={() => {}} />)
    await waitFor(() => expect(screen.getByText('Manual Tasks')).toBeInTheDocument())
  })
})
