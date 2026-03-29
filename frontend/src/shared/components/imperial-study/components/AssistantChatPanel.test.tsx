import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ChatMessage } from '../hooks/useAssistantPanes'
import { AssistantChatPanel } from './AssistantChatPanel'

beforeEach(() => {
  Element.prototype.scrollIntoView = vi.fn()
})

const messages: ChatMessage[] = [
  { id: '1', role: 'user', content: 'Hello', assistantType: 'chat', timestamp: Date.now() },
  {
    id: '2',
    role: 'assistant',
    content: 'Hi there',
    assistantType: 'chat',
    timestamp: Date.now(),
    streaming: true,
  },
]

describe('AssistantChatPanel', () => {
  it('renders message bubbles', () => {
    render(<AssistantChatPanel messages={messages} streaming={true} onClear={vi.fn()} />)
    expect(screen.getByText('Hello')).toBeInTheDocument()
    expect(screen.getByText('Hi there')).toBeInTheDocument()
  })

  it('shows streaming indicator for streaming message', () => {
    render(<AssistantChatPanel messages={messages} streaming={true} onClear={vi.fn()} />)
    const dot = document.querySelector('.is-chat-bubble__dot')
    expect(dot).toBeInTheDocument()
  })

  it('clear button calls onClear', async () => {
    const onClear = vi.fn()
    render(<AssistantChatPanel messages={messages} streaming={false} onClear={onClear} />)
    const clearBtn = document.querySelector('.is-chat-panel__clear')
    expect(clearBtn).toBeInTheDocument()
    await userEvent.click(clearBtn!)
    expect(onClear).toHaveBeenCalled()
  })

  it('renders error messages', () => {
    const errorMsg: ChatMessage[] = [
      {
        id: 'e1',
        role: 'assistant',
        content: '',
        assistantType: 'chat',
        timestamp: Date.now(),
        error: 'Failed',
      },
    ]
    render(<AssistantChatPanel messages={errorMsg} streaming={false} onClear={vi.fn()} />)
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })

  it('renders reasoning content', () => {
    const reasoningMsg: ChatMessage[] = [
      {
        id: 'r1',
        role: 'assistant',
        content: 'Answer',
        assistantType: 'chat',
        timestamp: Date.now(),
        reasoning: 'Thinking...',
      },
    ]
    render(<AssistantChatPanel messages={reasoningMsg} streaming={false} onClear={vi.fn()} />)
    expect(screen.getByText('Thinking...')).toBeInTheDocument()
    expect(screen.getByText('Answer')).toBeInTheDocument()
  })
})
