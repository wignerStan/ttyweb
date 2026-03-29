import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ConversationViewer } from './ConversationViewer'
import type { AISession, ConversationMessage } from './types'

const mockMessages: ConversationMessage[] = [
  { role: 'user', content: 'How do I run tests?', timestamp: '2026-01-01T10:00:00Z' },
  {
    role: 'assistant',
    content: 'Run `make test` from the project root.',
    timestamp: '2026-01-01T10:00:05Z',
  },
]

const mockSession: AISession = {
  id: 'sess-1',
  type: 'claude_code',
  model: 'claude-3-opus',
  title: 'Test Session',
  message_count: 2,
}

vi.mock('./useConversations', () => ({
  useConversation: () => ({
    messages: mockMessages,
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
}))

describe('ConversationViewer snapshot', () => {
  it('renders conversation with messages', () => {
    const { container } = render(
      <ConversationViewer sessionId="sess-1" sessionInfo={mockSession} onBack={vi.fn()} />,
    )
    expect(container).toMatchSnapshot()
  })
})
