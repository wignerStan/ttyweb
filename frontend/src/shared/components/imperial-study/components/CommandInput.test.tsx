import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { CommandInput } from './CommandInput'

vi.mock('../../VoiceInput', () => ({
  VoiceInput: ({
    onText,
    onPartial,
  }: {
    onText: (t: string) => void
    onPartial?: (t: string) => void
  }) => (
    <div>
      <button type="button" data-testid="voice-text" onClick={() => onText('hello voice')}>
        Voice Text
      </button>
      <button type="button" data-testid="voice-partial" onClick={() => onPartial?.('partial text')}>
        Voice Partial
      </button>
    </div>
  ),
}))

vi.mock('../../../../utils/auth', () => ({ getAuthHeader: () => 'Bearer tok' }))

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
    await userEvent.click(screen.getByTestId('voice-text'))
    expect(input).toHaveValue('say hello voice')
  })

  it('voice partial wraps in brackets', async () => {
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'say ')
    await userEvent.click(screen.getByTestId('voice-partial'))
    expect(input).toHaveValue('say [partial text]')
  })

  it('voice text to empty input sets text directly', async () => {
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.click(screen.getByTestId('voice-text'))
    expect(input).toHaveValue('hello voice')
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

  it('clicking a tag activates it', async () => {
    const onTagChange = vi.fn()
    render(<CommandInput activeTag={null} onTagChange={onTagChange} />)
    await userEvent.click(screen.getByText('Translate'))
    expect(onTagChange).toHaveBeenCalledWith('translator')
  })

  it('clicking active tag deactivates it', async () => {
    const onTagChange = vi.fn()
    render(<CommandInput activeTag="cli" onTagChange={onTagChange} />)
    await userEvent.click(screen.getByText('CLI'))
    expect(onTagChange).toHaveBeenCalledWith(null)
  })

  it('shows success feedback after dispatch', async () => {
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} onDispatched={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'test{Control>}{Enter}{/Control}')
    await waitFor(() => expect(screen.getByText(/Dispatched/)).toBeInTheDocument())
  })

  it('shows error feedback on failed dispatch', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: false, error: { message: 'No route found' } }),
      }),
    )
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} onDispatched={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'fail{Control>}{Enter}{/Control}')
    await waitFor(() => expect(screen.getByText('No route found')).toBeInTheDocument())
  })

  it('shows network error feedback on fetch exception', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('Network error')))
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} onDispatched={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'fail{Control>}{Enter}{/Control}')
    await waitFor(() => expect(screen.getByText('Network error')).toBeInTheDocument())
  })

  it('sends pane_target in payload when provided', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ success: true, data: { run_id: 'r2', task_id: 't2' } }),
    })
    vi.stubGlobal('fetch', fetchMock)
    render(
      <CommandInput
        activeTag={null}
        onTagChange={vi.fn()}
        onDispatched={vi.fn()}
        paneTarget="main:0:0"
      />,
    )
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'test{Control>}{Enter}{/Control}')
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
    const callArgs = JSON.parse(fetchMock.mock.calls[0]![1]!.body)
    expect(callArgs.params).toEqual({ pane_target: 'main:0:0' })
  })

  it('handles chat_fallback response by switching to chat mode', async () => {
    const onAssistantSend = vi.fn()
    const onTagChange = vi.fn()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            success: true,
            data: { run_id: 'r3', task_id: 't3', chat_fallback: true },
          }),
      }),
    )
    render(
      <CommandInput activeTag={null} onTagChange={onTagChange} onAssistantSend={onAssistantSend} />,
    )
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'hello{Control>}{Enter}{/Control}')
    await waitFor(() => expect(onAssistantSend).toHaveBeenCalledWith('hello', 'chat'))
    expect(onTagChange).toHaveBeenCalledWith('chat')
  })

  it('does not submit when input is empty', async () => {
    const onDispatched = vi.fn()
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} onDispatched={onDispatched} />)
    const sendBtn = document.querySelector('.is-command__send') as HTMLButtonElement
    expect(sendBtn.disabled).toBe(true)
    await userEvent.click(sendBtn)
    expect(onDispatched).not.toHaveBeenCalled()
  })

  it('active tag changes placeholder text', () => {
    render(<CommandInput activeTag="translator" onTagChange={vi.fn()} />)
    expect(screen.getByPlaceholderText(/Translate/)).toBeInTheDocument()
  })

  it('shows loading state in send button during dispatch', async () => {
    let resolveFetch: (v: unknown) => void
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveFetch = resolve
          }),
      ),
    )
    render(<CommandInput activeTag={null} onTagChange={vi.fn()} onDispatched={vi.fn()} />)
    const input = screen.getByPlaceholderText(/Enter command/)
    await userEvent.type(input, 'test{Control>}{Enter}{/Control}')
    await waitFor(() => {
      const sendBtn = document.querySelector('.is-command__send')
      expect(sendBtn?.className).toContain('loading')
    })
    resolveFetch!({
      ok: true,
      json: () => Promise.resolve({ success: true, data: { run_id: 'r1' } }),
    })
    await waitFor(() => {
      const sendBtn = document.querySelector('.is-command__send')
      expect(sendBtn?.className).not.toContain('loading')
    })
  })
})
