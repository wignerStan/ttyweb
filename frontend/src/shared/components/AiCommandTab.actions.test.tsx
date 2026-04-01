import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { AiCommandTab } from './AiCommandTab'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
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

describe('AiCommandTab actions', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) }))
    vi.stubGlobal('WebSocket', vi.fn().mockImplementation(createMockWs))
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sends input directly to terminal', async () => {
    const onSend = vi.fn()
    renderWithProviders(<AiCommandTab onSend={onSend} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'ls -la')
    const sendBtns = screen
      .getAllByRole('button')
      .filter((btn) => btn.textContent?.includes('\\u53D1\\u9001\\u7EC8\\u7AEF'))
    expect(sendBtns[0]).toBeTruthy()
    await userEvent.click(sendBtns[0]!)

    expect(onSend).toHaveBeenCalledWith('ls -la\n')
  })

  it('clears input and result', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'echo hi', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'say hi')
    await userEvent.click(screen.getByRole('button', { name: /AI/ }))
    await waitFor(() => expect(screen.getByText('echo hi')).toBeInTheDocument())

    const clearBtns = screen
      .getAllByRole('button')
      .filter((btn) => btn.getAttribute('title')?.includes('\\u6E05\\u7A7A'))
    expect(clearBtns[0]).toBeTruthy()
    await userEvent.click(clearBtns[0]!)

    expect(screen.queryByText('echo hi')).not.toBeInTheDocument()
  })

  it('accepts initialText prop', async () => {
    const onTextConsumed = vi.fn()
    renderWithProviders(
      <AiCommandTab onSend={() => {}} initialText="pre-filled" onTextConsumed={onTextConsumed} />,
    )
    await waitFor(() => expect(screen.getByDisplayValue('pre-filled')).toBeInTheDocument())
    expect(onTextConsumed).toHaveBeenCalled()
  })

  it('shows system prompt toggle', async () => {
    renderWithProviders(<AiCommandTab onSend={() => {}} />)
    const promptBtns = screen
      .getAllByRole('button')
      .filter((btn) => btn.textContent?.includes('\\u7CFB\\u7EDF\\u63D0\\u793A\\u8BCD'))
    expect(promptBtns.length).toBeGreaterThan(0)
  })

  it('generates via Enter key', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ command: 'pwd', explanation: '' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<AiCommandTab onSend={() => {}} />)

    const textarea = document.querySelector('textarea')!
    await userEvent.type(textarea, 'show directory{Enter}')

    await waitFor(() => expect(fetchMock).toHaveBeenCalled())
  })
})
