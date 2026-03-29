import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { SnippetsTab } from './SnippetsTab'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const noop = () => {}

const snippets = [
  { name: 'ls', command: 'ls -la' },
  { name: 'git st', command: 'git status' },
]

function mockFetchJSON(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(data) })
}

describe('SnippetsTab', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetchJSON({ snippets }))
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders snippet list', async () => {
    renderWithProviders(<SnippetsTab onSend={noop} />)
    await waitFor(() => expect(screen.getByText('ls')).toBeInTheDocument())
    expect(screen.getByText('git st')).toBeInTheDocument()
  })

  it('shows empty state when no snippets', async () => {
    vi.stubGlobal('fetch', mockFetchJSON({ snippets: [] }))
    renderWithProviders(<SnippetsTab onSend={noop} />)
    await waitFor(() => expect(screen.getByText(/暂无片段/)).toBeInTheDocument())
  })

  it('creates a snippet', async () => {
    const fetchMock = mockFetchJSON({ snippets })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ snippets }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ snippets }) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<SnippetsTab onSend={noop} />)
    await userEvent.click(screen.getByText('添加'))
    await userEvent.type(screen.getByPlaceholderText('名称'), 'deploy')
    await userEvent.type(screen.getByPlaceholderText('命令'), 'make deploy')
    await userEvent.click(screen.getByText('保存'))
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/snippets',
        expect.objectContaining({ method: 'POST' }),
      ),
    )
  })

  it('deletes a snippet', async () => {
    const fetchMock = mockFetchJSON({ snippets })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ snippets }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ snippets: [] }) })
    vi.stubGlobal('fetch', fetchMock)
    renderWithProviders(<SnippetsTab onSend={noop} />)
    await waitFor(() => expect(screen.getByText('ls')).toBeInTheDocument())
    const trashButtons = document.querySelectorAll('button svg.lucide-trash-2')
    if (trashButtons.length > 0) {
      const trashBtn = trashButtons[0]!.closest('button')!
      await userEvent.click(trashBtn)
    }
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
  })

  it('sends snippet command to terminal', async () => {
    const onSend = vi.fn()
    renderWithProviders(<SnippetsTab onSend={onSend} />)
    await waitFor(() => expect(screen.getByText('ls')).toBeInTheDocument())
    const playButtons = screen
      .getAllByRole('button')
      .filter((btn) => btn.innerHTML.includes('play'))
    if (playButtons.length > 0) {
      expect(playButtons[0]).toBeTruthy()
      await userEvent.click(playButtons[0]!)
    }
    await waitFor(() => expect(onSend).toHaveBeenCalledWith('ls -la\n'))
  })
})
