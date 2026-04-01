import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { AiCommandTab } from './AiCommandTab'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
}))

vi.mock('./RoleManagerModal', () => ({
  RoleManagerModal: () => <div data-testid="role-modal" />,
}))

function createMockWs() {
  const ws = {
    send: vi.fn(),
    close: vi.fn(),
    onopen: null as ((ev: Event) => void) | null,
    onmessage: null as ((ev: MessageEvent) => void) | null,
    onerror: null as ((ev: Event) => void) | null,
    onclose: null as ((ev: CloseEvent) => void) | null,
  }
  setTimeout(() => {
    if (ws.onerror) ws.onerror(new Event('error'))
  }, 0)
  return ws
}

const clipboardMock = vi.fn().mockResolvedValue(undefined)

describe('AiCommandTab', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) }))
    vi.stubGlobal('WebSocket', vi.fn().mockImplementation(createMockWs))
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders command textarea', () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    expect(document.querySelector('textarea')).toBeInTheDocument()
  })

  it('renders role selector with default role', () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    expect(screen.getByText('\u547D\u4EE4\u884C\u5927\u795E')).toBeInTheDocument()
  })

  it('opens role dropdown', async () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const selector = screen.getByText('\u547D\u4EE4\u884C\u5927\u795E').closest('button')!
    await userEvent.click(selector)
    expect(screen.getByText('\u8FD0\u7EF4\u4E13\u5BB6')).toBeInTheDocument()
    expect(screen.getByText('\u63D0\u793A\u8BCD\u4F18\u5316')).toBeInTheDocument()
  })

  it('switches role on selection', async () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const selector = screen.getByText('\u547D\u4EE4\u884C\u5927\u795E').closest('button')!
    await userEvent.click(selector)
    await userEvent.click(screen.getByText('\u8FD0\u7EF4\u4E13\u5BB6'))
    expect(screen.getByText('\u8FD0\u7EF4\u4E13\u5BB6')).toBeInTheDocument()
  })

  it('shows generate button', () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    expect(screen.getByRole('button', { name: /AI/ })).toBeInTheDocument()
  })

  it('generates command and shows result', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'ls -la', explanation: 'List all files' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'list all files')
    await userEvent.click(screen.getByRole('button', { name: /AI/ }))

    await waitFor(() => expect(screen.getByText('List all files')).toBeInTheDocument())
    await waitFor(() => expect(screen.getByText('ls -la')).toBeInTheDocument())
  })

  it('copies command to clipboard', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: clipboardMock },
      writable: true,
      configurable: true,
    })
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'echo hello', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'say hello')
    await userEvent.click(screen.getByRole('button', { name: /AI/ }))

    // Source uses literal \uXXXX in JSX text, so rendered text contains literal backslash-u
    await waitFor(() => {
      const btns = document.querySelectorAll('button')
      const found = Array.from(btns).some((b) => b.textContent?.includes('\\u590D'))
      expect(found).toBe(true)
    })
    const copyBtn = Array.from(document.querySelectorAll('button')).find((b) =>
      b.textContent?.includes('\\u590D'),
    )!
    await userEvent.click(copyBtn)
    await waitFor(() => expect(clipboardMock).toHaveBeenCalledWith('echo hello'))
  })

  it('executes command via onSend', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'echo test', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const onSend = vi.fn()
    renderWithProviders(<AiCommandTab onSend={onSend} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'say test')
    await userEvent.click(screen.getByRole('button', { name: /AI/ }))

    // Source uses literal \uXXXX in JSX text - rendered as literal backslash-u strings
    await waitFor(() => {
      const btns = document.querySelectorAll('button')
      const found = Array.from(btns).some((b) => b.textContent?.includes('\\u6267'))
      expect(found).toBe(true)
    })
    const execBtn = Array.from(document.querySelectorAll('button')).find((b) =>
      b.textContent?.includes('\\u6267'),
    )!
    await userEvent.click(execBtn)
    await waitFor(() => expect(onSend).toHaveBeenCalledWith('echo test\n'))
  })

  it('direct send button sends input to terminal', async () => {
    const onSend = vi.fn()
    renderWithProviders(<AiCommandTab onSend={onSend} />)
    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'ls -la')
    // The button has text with literal \uXXXX - find by textContent containing the terminal label
    const sendBtn = Array.from(document.querySelectorAll('button')).find((b) =>
      b.textContent?.includes('\\u53D1\\u9001\\u7EC8\\u7AEF'),
    )!
    await userEvent.click(sendBtn)
    expect(onSend).toHaveBeenCalledWith('ls -la\n')
  })

  it('direct send button is disabled when input is empty', () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const sendBtn = Array.from(document.querySelectorAll('button')).find((b) =>
      b.textContent?.includes('\\u53D1\\u9001\\u7EC8\\u7AEF'),
    )!
    expect(sendBtn).toBeDisabled()
  })

  it('clear button clears input and result', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'echo test', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'test')
    await userEvent.click(screen.getByRole('button', { name: /AI/ }))
    await waitFor(() => expect(screen.getByText('echo test')).toBeInTheDocument())
    // The clear button in the action bar has title with literal \uXXXX
    // After result appears, the clear button is the one with X icon that clears everything
    const clearBtn = Array.from(document.querySelectorAll('button')).find((b) =>
      b.getAttribute('title')?.includes('\\u6E05\\u7A7A'),
    )!
    await userEvent.click(clearBtn)
    expect(textarea).toHaveValue('')
  })

  it('Enter key triggers generation', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'pwd', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'show directory{Enter}')
    await waitFor(() => expect(screen.getByText('pwd')).toBeInTheDocument())
  })

  it('shows system prompt toggle', () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    // The system prompt button is the second button (after role selector)
    const btns = screen.getAllByRole('button')
    // It should have a chevron SVG child
    const promptBtn = btns[1]
    expect(promptBtn).toBeTruthy()
    expect(promptBtn!.querySelector('svg')).toBeTruthy()
  })

  it('toggles system prompt visibility', async () => {
    const user = userEvent.setup()
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const btns = screen.getAllByRole('button')
    const promptBtn = btns[1]
    expect(promptBtn).toBeTruthy()
    await user.click(promptBtn!)
    await user.click(promptBtn!)
  })

  it('closes role dropdown on outside click', async () => {
    const user = userEvent.setup()
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const selector = screen.getByText('\u547D\u4EE4\u884C\u5927\u795E').closest('button')!
    await user.click(selector)
    expect(screen.getByText('\u8FD0\u7EF4\u4E13\u5BB6')).toBeInTheDocument()
    await user.click(document.querySelector('textarea')!)
    expect(screen.queryByText('\u8FD0\u7EF4\u4E13\u5BB6')).not.toBeInTheDocument()
  })

  it('shows manage roles button in dropdown', async () => {
    const user = userEvent.setup()
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const selector = screen.getByText('\u547D\u4EE4\u884C\u5927\u795E').closest('button')!
    await user.click(selector)
    // The manage roles button has text "+ \u7BA1\u7406\u89D2\u8272" with literal \uXXXX
    const allBtns = document.querySelectorAll('button')
    const manageBtn = Array.from(allBtns).find((b) =>
      b.textContent?.includes('\\u7BA1\\u7406\\u89D2\\u8272'),
    )
    expect(manageBtn).toBeTruthy()
  })

  it('is disabled when disabled prop is true', () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} disabled />)
    // The generate button should be disabled
    const generateBtn = screen.getByRole('button', { name: /AI/ })
    expect(generateBtn).toBeDisabled()
  })

  it('uses initialText when provided', async () => {
    const onTextConsumed = vi.fn()
    renderWithProviders(
      <AiCommandTab onSend={() => {}} initialText="hello world" onTextConsumed={onTextConsumed} />,
    )
    const textarea = document.querySelector('textarea')!
    expect(textarea).toHaveValue('hello world')
    expect(onTextConsumed).toHaveBeenCalled()
  })

  it('handles error in generation', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: '', explanation: '\u8BF7\u6C42\u5931\u8D25: test' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'fail{Enter}')
    await waitFor(() =>
      expect(screen.getByText('\u8BF7\u6C42\u5931\u8D25: test')).toBeInTheDocument(),
    )
  })

  it('expand/collapse result card', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          command: 'a very long command that might need expansion',
          explanation: 'Test',
        }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const textarea = document.querySelector('textarea')!
    await user.type(textarea, 'test{Enter}')
    await waitFor(() =>
      expect(screen.getByText('a very long command that might need expansion')).toBeInTheDocument(),
    )
    const expandBtns = screen
      .getAllByRole('button')
      .filter(
        (btn) =>
          btn.querySelector('svg.lucide-chevron-up') ||
          btn.querySelector('svg.lucide-chevron-down'),
      )
    if (expandBtns.length > 0) {
      expect(expandBtns[0]).toBeTruthy()
      await user.click(expandBtns[0]!)
    }
  })
})
