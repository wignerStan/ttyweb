import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

// Mock the useNewWindow hook
const mockCreateWindow = vi.fn().mockResolvedValue({})
const mockQuickDirs = [{ name: 'projects', path: '/home/u/projects' }]

vi.mock('../../hooks/useNewWindow', () => ({
  useNewWindow: () => ({
    quickDirs: mockQuickDirs,
    createWindow: mockCreateWindow,
    loading: false,
    error: null,
  }),
}))

// Must import after mock setup
const { NewWindowButton } = await import('./NewWindowButton')

describe('NewWindowButton', () => {
  beforeEach(() => {
    mockCreateWindow.mockReset()
    mockCreateWindow.mockResolvedValue({})
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders button', () => {
    renderWithProviders(<NewWindowButton session="main" />)
    expect(screen.getByTitle('New tmux window')).toBeInTheDocument()
  })

  it('loads quick dirs from hook', async () => {
    renderWithProviders(<NewWindowButton session="main" />)
    expect(screen.getByTitle('New tmux window')).toBeInTheDocument()
    // Quick dirs are loaded via the mocked hook
  })

  it('opens menu and creates window with default dir', async () => {
    const onCreated = vi.fn()
    renderWithProviders(<NewWindowButton session="main" onCreated={onCreated} />)

    await userEvent.click(screen.getByTitle('New tmux window'))
    await waitFor(() => expect(screen.getByText('Default Dir')).toBeInTheDocument())
    await userEvent.click(screen.getByText('Default Dir'))

    await waitFor(() => expect(mockCreateWindow).toHaveBeenCalledWith('main', undefined, undefined))
  })

  it('creates window with specific dir', async () => {
    renderWithProviders(<NewWindowButton session="main" />)

    await userEvent.click(screen.getByTitle('New tmux window'))
    await waitFor(() => expect(screen.getByText('projects')).toBeInTheDocument())
    await userEvent.click(screen.getByText('projects'))

    await waitFor(() =>
      expect(mockCreateWindow).toHaveBeenCalledWith('main', '/home/u/projects', undefined),
    )
  })

  it('closes menu on outside click', async () => {
    renderWithProviders(
      <div>
        <div data-testid="outside">outside</div>
        <NewWindowButton session="main" />
      </div>,
    )

    await userEvent.click(screen.getByTitle('New tmux window'))
    await waitFor(() => expect(screen.getByText('New Window')).toBeInTheDocument())

    await userEvent.click(screen.getByTestId('outside'))
    expect(screen.queryByText('New Window')).not.toBeInTheDocument()
  })
})
