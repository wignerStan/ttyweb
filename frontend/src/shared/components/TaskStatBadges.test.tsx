import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderWithProviders, screen } from '../../test-utils'
import { TaskStatBadges } from './TaskStatBadges'

function mockFetchResponse(data: Record<string, unknown>) {
  return vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(data),
  } as Response)
}

describe('TaskStatBadges', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = mockFetchResponse({
      tasks: [
        { task_status: 'in_progress' },
        { task_status: 'completed' },
        { task_status: 'completed' },
        { task_status: 'failed' },
        { task_status: 'waiting' },
      ],
    })
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('localStorage', { getItem: () => null })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders "in progress" and "completed" badges always', async () => {
    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    expect(screen.getByText('进行中')).toBeInTheDocument()
    expect(screen.getByText('已完成')).toBeInTheDocument()
  })

  it('shows "failed" badge when failed > 0', async () => {
    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    const failedBadge = document.querySelector('.border-error\\/20')
    expect(failedBadge).toBeInTheDocument()
    expect(failedBadge?.querySelector('.font-mono')?.textContent).toBe('1')
  })

  it('hides "failed" badge when failed is 0', async () => {
    fetchMock = mockFetchResponse({
      tasks: [{ task_status: 'in_progress' }, { task_status: 'completed' }],
    })
    vi.stubGlobal('fetch', fetchMock)

    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    expect(screen.queryByText('Failed')).not.toBeInTheDocument()
  })

  it('shows "waiting" badge when waiting > 0', async () => {
    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    const waitingBadge = document.querySelector('.border-warning\\/20')
    expect(waitingBadge).toBeInTheDocument()
    expect(waitingBadge?.querySelector('.font-mono')?.textContent).toBe('1')
  })

  it('hides "waiting" badge when waiting is 0', async () => {
    fetchMock = mockFetchResponse({
      tasks: [{ task_status: 'in_progress' }],
    })
    vi.stubGlobal('fetch', fetchMock)

    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    expect(screen.queryByText('Waiting')).not.toBeInTheDocument()
  })

  it('refresh button triggers re-fetch', async () => {
    const user = await import('@testing-library/user-event')
    const userEvent = user.default.setup()

    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)

    const refreshBtn = screen.getByTitle('Refresh')
    await userEvent.click(refreshBtn)

    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('handles error gracefully without crashing', async () => {
    fetchMock = vi.fn().mockRejectedValue(new Error('Network error'))
    vi.stubGlobal('fetch', fetchMock)

    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    expect(screen.getByText('进行中')).toBeInTheDocument()
    expect(screen.getByText('已完成')).toBeInTheDocument()
  })

  it('handles non-ok response gracefully', async () => {
    fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({}),
    } as Response)
    vi.stubGlobal('fetch', fetchMock)

    await act(async () => {
      renderWithProviders(<TaskStatBadges />)
    })
    expect(screen.getByText('进行中')).toBeInTheDocument()
  })
})
