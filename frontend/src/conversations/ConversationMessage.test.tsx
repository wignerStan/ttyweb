import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { ConversationMessage } from './ConversationMessage'
import type { ConversationMessage as MsgType } from './types'

describe('ConversationMessage', () => {
  it('user messages styled correctly', () => {
    const msg: MsgType = {
      role: 'user',
      content: 'Hello assistant',
      timestamp: '2026-03-29T12:00:00Z',
    }
    renderWithProviders(<ConversationMessage message={msg} />)
    const el = document.querySelector('.conv-msg--user')
    expect(el).toBeInTheDocument()
    expect(screen.getByText('Hello assistant')).toBeInTheDocument()
  })

  it('assistant messages styled correctly', () => {
    const msg: MsgType = {
      role: 'assistant',
      content: 'I can help with that',
      timestamp: '2026-03-29T12:01:00Z',
    }
    renderWithProviders(<ConversationMessage message={msg} />)
    const el = document.querySelector('.conv-msg--assistant')
    expect(el).toBeInTheDocument()
    expect(screen.getByText('I can help with that')).toBeInTheDocument()
  })

  it('shows timestamp', () => {
    const msg: MsgType = {
      role: 'user',
      content: 'test',
      timestamp: '2026-03-29T12:00:00Z',
    }
    renderWithProviders(<ConversationMessage message={msg} />)
    const ts = document.querySelector('.conv-msg-timestamp')
    expect(ts).toBeInTheDocument()
  })

  it('tool result inline rendering', () => {
    const msg: MsgType = {
      role: 'assistant',
      content: 'Running command',
      timestamp: '',
      tool_use: [
        {
          id: 'tu1',
          name: 'Bash',
          input: { command: 'ls -la' },
        },
      ],
      tool_result: [
        {
          tool_use_id: 'tu1',
          output: 'file1.txt\nfile2.txt',
        },
      ],
    }
    renderWithProviders(<ConversationMessage message={msg} />)
    expect(screen.getByText('Bash')).toBeInTheDocument()
  })
})
