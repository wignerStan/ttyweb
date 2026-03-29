import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import { LoginModal } from './LoginModal'

const mockLogin = vi.fn()

vi.mock('../../utils/auth', () => ({
  login: (...args: unknown[]) => mockLogin(...args),
}))

describe('LoginModal', () => {
  beforeEach(() => {
    mockLogin.mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders username/password fields and submit button', () => {
    renderWithProviders(<LoginModal onLogin={() => {}} />)
    expect(screen.getByLabelText('Username')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument()
  })

  it('disables submit when fields are empty', () => {
    renderWithProviders(<LoginModal onLogin={() => {}} />)
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeDisabled()
  })

  it('shows loading state while authenticating', async () => {
    mockLogin.mockReturnValue(new Promise(() => {}))
    renderWithProviders(<LoginModal onLogin={() => {}} />)

    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'pass')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    await waitFor(() => expect(screen.getByText('Authenticating...')).toBeInTheDocument())
  })

  it('shows error on failure', async () => {
    mockLogin.mockResolvedValue({ success: false, error: 'Bad credentials' })
    renderWithProviders(<LoginModal onLogin={() => {}} />)

    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'wrong')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    await waitFor(() => expect(screen.getByText('Bad credentials')).toBeInTheDocument())
  })

  it('calls onLogin on success', async () => {
    const onLogin = vi.fn()
    mockLogin.mockResolvedValue({ success: true })
    renderWithProviders(<LoginModal onLogin={onLogin} />)

    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'pass')
    await userEvent.click(screen.getByRole('button', { name: 'Sign In' }))

    await waitFor(() => expect(onLogin).toHaveBeenCalled())
  })

  it('submits on Enter key', async () => {
    mockLogin.mockResolvedValue({ success: true })
    const onLogin = vi.fn()
    renderWithProviders(<LoginModal onLogin={onLogin} />)

    await userEvent.type(screen.getByLabelText('Username'), 'admin')
    await userEvent.type(screen.getByLabelText('Password'), 'pass{Enter}')

    await waitFor(() => expect(onLogin).toHaveBeenCalled())
  })
})
