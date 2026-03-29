import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AISession } from './types'

const mockSessions: AISession[] = [
  {
    id: 's1',
    type: 'claude_code',
    model: 'claude-3-opus',
    title: 'Fix authentication',
    message_count: 12,
    timestamp: new Date().toISOString(),
  },
  {
    id: 's2',
    type: 'codex',
    model: 'gpt-4',
    title: 'Refactor API',
    message_count: 5,
    timestamp: new Date(Date.now() - 3600000).toISOString(),
  },
]

let sessions = mockSessions
let loading = false
let error = ''
const refetch = vi.fn()

vi.mock('./useConversations', () => ({
  useConversations: () => ({
    sessions,
    loading,
    error,
    refetch,
  }),
}))

import { renderWithProviders } from '../test-utils'
import { ConversationList } from './ConversationList'

describe('ConversationList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    sessions = mockSessions
    loading = false
    error = ''
  })

  it('renders list of AI sessions', () => {
    renderWithProviders(<ConversationList selectedSessionId={null} onSelectSession={vi.fn()} />)
    expect(screen.getByText('Fix authentication')).toBeInTheDocument()
    expect(screen.getByText('Refactor API')).toBeInTheDocument()
  })

  it('loading state', () => {
    sessions = []
    loading = true
    renderWithProviders(<ConversationList selectedSessionId={null} onSelectSession={vi.fn()} />)
    expect(screen.getByText(/Loading.../)).toBeInTheDocument()
  })

  it('error state', () => {
    sessions = []
    error = 'Network error'
    renderWithProviders(<ConversationList selectedSessionId={null} onSelectSession={vi.fn()} />)
    expect(screen.getByText('Network error')).toBeInTheDocument()
  })

  it('clicking session calls onSelectSession', async () => {
    const onSelectSession = vi.fn()
    renderWithProviders(
      <ConversationList selectedSessionId={null} onSelectSession={onSelectSession} />,
    )
    const user = userEvent.setup()
    await user.click(screen.getByText('Fix authentication'))
    expect(onSelectSession).toHaveBeenCalledOnce()
    expect(onSelectSession.mock.calls[0][0].id).toBe('s1')
  })

  it('empty state', () => {
    sessions = []
    renderWithProviders(<ConversationList selectedSessionId={null} onSelectSession={vi.fn()} />)
    expect(screen.getByText('No conversations found')).toBeInTheDocument()
  })
})
