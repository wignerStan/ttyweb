import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { CommandInput } from './CommandInput'

vi.mock('../../VoiceInput', () => ({
  VoiceInput: ({ onText }: { onText: (t: string) => void }) => (
    <button data-testid="voice" onClick={() => onText('hello voice')}>
      Voice
    </button>
  ),
}))

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => '' }))

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ success: true, data: { run_id: 'r1', task_id: 't1' } }),
    }),
  )
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('CommandInput', () => {
  it('renders input field and send button', () => {
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} />)
    expect(screen.getByPlaceholderText(/Enter command/)).toBeInTheDocument()
    const sendBtn = document.querySelector('.is-command__send')
    expect(sendBtn).toBeInTheDocument()
  })

  it('send button triggers dispatch on Ctrl+Enter', async () => {
    const onDispatched = vi.fn()
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} onDispatched={onDispatched} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'do something{Control>}{Enter}{/Control}')
    await waitFor(() => expect(onDispatched).toHaveBeenCalled())
  })

  it('voice button appends text', async () => {
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'say ')
    await userEvent.click(screen.getByTestId('voice'))
    expect(input).toHaveValue('say hello voice')
  })

  it('assistant tag routes to onAssistantSend', async () => {
    const onAssistantSend = vi.fn()
    render(
      <CommandInput activeTag="chat" onTagChange={vi.fn()} onAssistantSend={onAssistantSend} />,
    )
    const input = screen.getByPlaceholderText(/Chat/)
    await userEvent.type(input, 'hello{Control>}{Enter}{/Control}')
    await waitFor(() => expect(onAssistantSend).toHaveBeenCalledWith('hello', 'chat'))
  })

  it('renders assistant tag buttons', () => {
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} />)
    expect(screen.getByText('Translate')).toBeInTheDocument()
    expect(screen.getByText('CLI')).toBeInTheDocument()
    expect(screen.getByText('Market')).toBeInTheDocument()
    expect(screen.getByText('Chat')).toBeInTheDocument()
  })
})
