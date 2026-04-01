import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { mockFetchHttpError } from '../../test-helpers'
import { renderWithProviders } from '../../test-utils'
import { ConfigViewer } from './ConfigViewer'

vi.mock('../../utils/auth', () => ({
  getAuthHeader: () => 'Bearer test-token',
  getAuthHeaders: () => ({ Authorization: 'Bearer test-token' }),
}))

const configData = {
  opencode: { content: { key: 'val' }, path: '/home/.config/opencode.json' },
  oh_my_opencode: { content: null, path: '/home/.config/oh-my-opencode.json' },
}

describe('ConfigViewer', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(configData) }),
    )
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows loading state', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => new Promise(() => {})),
    )
    renderWithProviders(<ConfigViewer />)
    expect(screen.getByText('加载配置...')).toBeInTheDocument()
  })

  it('shows error state on fetch failure', async () => {
    vi.stubGlobal('fetch', mockFetchHttpError(500))
    renderWithProviders(<ConfigViewer />)
    await waitFor(() => expect(screen.getByText(/加载失败/)).toBeInTheDocument())
  })

  it('renders tab buttons on success', async () => {
    renderWithProviders(<ConfigViewer />)
    await waitFor(() => expect(screen.getByText('opencode.json')).toBeInTheDocument())
    expect(screen.getByText('oh-my-opencode.json')).toBeInTheDocument()
  })

  it('switches tabs', async () => {
    renderWithProviders(<ConfigViewer />)
    await waitFor(() => expect(screen.getByText('opencode.json')).toBeInTheDocument())
    await userEvent.click(screen.getByText('oh-my-opencode.json'))
    expect(screen.getByText('无数据')).toBeInTheDocument()
  })

  it('shows missing file message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            opencode: { content: null, path: '/home/a.json', missing: true },
          }),
      }),
    )
    renderWithProviders(<ConfigViewer />)
    await waitFor(() => expect(screen.getByText('文件不存在')).toBeInTheDocument())
  })
})
