import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { Sidebar } from '../components/Sidebar'

const mockSessions = [
  { name: 'session-1', windows: 1, attached: false },
  { name: 'session-2', windows: 2, attached: true },
]

const mockDetail = {
  name: 'session-1',
  windows: 1,
  attached: false,
  panes: [{ id: '%0', title: 'bash', current_command: 'vim', running: true }],
}

function mockFetchSuccess(data: unknown) {
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ success: true, data }),
  })
}

describe('Sidebar', () => {
  let onSelect: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onSelect = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders Sessions header and new session input', () => {
    globalThis.fetch = mockFetchSuccess(mockSessions) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    expect(screen.getByText('Sessions')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('new session')).toBeInTheDocument()
  })

  it('fetches sessions on mount', async () => {
    const fetchMock = mockFetchSuccess(mockSessions)
    globalThis.fetch = fetchMock as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/sessions')
    })
  })

  it('displays sessions from API', async () => {
    globalThis.fetch = mockFetchSuccess(mockSessions) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    await waitFor(() => {
      expect(screen.getByText('session-1')).toBeInTheDocument()
    })
    expect(screen.getByText('session-2')).toBeInTheDocument()
  })

  it('shows empty state when no sessions', async () => {
    globalThis.fetch = mockFetchSuccess([]) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    await waitFor(() => {
      expect(screen.getByText('No sessions found')).toBeInTheDocument()
    })
  })

  it('shows error message when fetch fails', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ success: false, error: 'connection failed' }),
    }) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    await waitFor(() => {
      expect(screen.getByText('connection failed')).toBeInTheDocument()
    })
  })

  it('allows typing in the new session input', async () => {
    const user = userEvent.setup()
    globalThis.fetch = mockFetchSuccess([]) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    const input = screen.getByPlaceholderText('new session')
    await user.type(input, 'my-new-session')

    expect(input).toHaveValue('my-new-session')
  })

  it('creates session via POST when Enter pressed in input', async () => {
    const user = userEvent.setup()
    const listMock = mockFetchSuccess(mockSessions)
    globalThis.fetch = vi.fn((_url: string | URL | Request) => {
      return listMock() as ReturnType<typeof fetch>
    }) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    const input = screen.getByPlaceholderText('new session')
    await user.type(input, 'new-session-test')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      const calls = (globalThis.fetch as unknown as ReturnType<typeof vi.fn>).mock.calls
      const postCall = calls.find(
        (c: unknown[]) => c[1] && (c[1] as Record<string, unknown>).method === 'POST',
      )
      expect(postCall).toBeDefined()
      expect(postCall![0]).toBe('/api/sessions')
    })
  })

  it('shows attached badge for attached sessions', async () => {
    globalThis.fetch = mockFetchSuccess(mockSessions) as unknown as typeof fetch

    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    await waitFor(() => {
      expect(screen.getByText('A')).toBeInTheDocument()
    })
  })

  it('expands session on click and shows pane details', async () => {
    const listMock = mockFetchSuccess(mockSessions)
    const detailMock = mockFetchSuccess(mockDetail)
    globalThis.fetch = vi.fn((_url: string | URL | Request) => {
      if (String(_url).includes('sessions/session-1')) {
        return detailMock()
      }
      return listMock()
    }) as unknown as typeof fetch

    const user = userEvent.setup()
    render(<Sidebar onSelect={onSelect as (session: string, pane?: string) => void} />)

    await waitFor(() => {
      expect(screen.getByText('session-1')).toBeInTheDocument()
    })

    await user.click(screen.getByText('session-1'))

    await waitFor(() => {
      expect(screen.getByText('Connect to session')).toBeInTheDocument()
    })
  })
})
