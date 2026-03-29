import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { AiCommandTab } from './AiCommandTab'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

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

describe('AiCommandTab', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) }))
    vi.stubGlobal('WebSocket', vi.fn().mockImplementation(createMockWs))
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
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
    expect(screen.getByText('命令行大神')).toBeInTheDocument()
  })

  it('opens role dropdown', async () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const selector = screen.getByText('命令行大神').closest('button')!
    await userEvent.click(selector)
    expect(screen.getByText('运维专家')).toBeInTheDocument()
    expect(screen.getByText('提示词优化')).toBeInTheDocument()
  })

  it('switches role on selection', async () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const selector = screen.getByText('命令行大神').closest('button')!
    await userEvent.click(selector)
    await userEvent.click(screen.getByText('运维专家'))
    expect(screen.getByText('运维专家')).toBeInTheDocument()
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
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'echo hello', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'say hello')
    await userEvent.click(screen.getByRole('button', { name: /AI/ }))

    await waitFor(() => {
      const copyBtns = screen
        .getAllByRole('button')
        .filter((btn) => btn.textContent?.includes('\\u590D\\u5236'))
      expect(copyBtns.length).toBeGreaterThan(0)
    })
    const copyBtns = screen
      .getAllByRole('button')
      .filter((btn) => btn.textContent?.includes('\\u590D\\u5236'))
    await userEvent.click(copyBtns[0])
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('echo hello'))
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

    await waitFor(() => {
      const execBtns = screen
        .getAllByRole('button')
        .filter((btn) => btn.textContent?.includes('\\u6267\\u884C'))
      expect(execBtns.length).toBeGreaterThan(0)
    })
    const execBtns = screen
      .getAllByRole('button')
      .filter((btn) => btn.textContent?.includes('\\u6267\\u884C'))
    await userEvent.click(execBtns[0])
    await waitFor(() => expect(onSend).toHaveBeenCalledWith('echo test\n'))
  })
})
