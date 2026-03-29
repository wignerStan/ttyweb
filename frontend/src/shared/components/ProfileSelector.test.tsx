import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { Profile } from '../../types'
import { ProfileSelector } from './ProfileSelector'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const profiles: Profile[] = [
  { id: 1, profile_key: 'default', name: 'Default', sort_order: 0 },
  { id: 2, profile_key: 'dev', name: 'Dev', sort_order: 1 },
]

const noop = () => {}

function mockFetchJSON(data: unknown) {
  return vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(data) })
}

describe('ProfileSelector', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetchJSON({ profiles }))
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders current profile name', async () => {
    renderWithProviders(<ProfileSelector currentProfile={profiles[0]} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
  })

  it('opens and closes dropdown', async () => {
    renderWithProviders(<ProfileSelector currentProfile={profiles[0]} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = document.querySelector('.profile-current')!
    await userEvent.click(header)
    expect(screen.getByText('Dev')).toBeInTheDocument()
    await userEvent.click(screen.getByText('Dev'))
    expect(screen.queryByText('New Profile')).not.toBeInTheDocument()
  })

  it('lists profiles from API', async () => {
    const onChange = vi.fn()
    renderWithProviders(<ProfileSelector currentProfile={null} onProfileChange={onChange} />)
    await waitFor(() => expect(onChange).toHaveBeenCalled())
    expect(onChange).toHaveBeenCalledWith(profiles[0])
  })

  it('selects a profile', async () => {
    const onChange = vi.fn()
    renderWithProviders(<ProfileSelector currentProfile={profiles[0]} onProfileChange={onChange} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = document.querySelector('.profile-current')!
    await userEvent.click(header)
    await userEvent.click(screen.getByText('Dev'))
    expect(onChange).toHaveBeenCalledWith(profiles[1])
  })

  it('creates a new profile', async () => {
    const fetchMock = mockFetchJSON({ profiles })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ profiles }) })
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ id: 3, profile_key: 'new', name: 'New' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const onChange = vi.fn()
    renderWithProviders(<ProfileSelector currentProfile={profiles[0]} onProfileChange={onChange} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = document.querySelector('.profile-current')!
    await userEvent.click(header)
    await waitFor(() => expect(screen.getByText('New Profile')).toBeInTheDocument())
    await userEvent.click(screen.getByText('New Profile'))
    const input = screen.getByPlaceholderText('Profile name...')
    await userEvent.type(input, 'New')
    await userEvent.click(document.querySelector('.btn-confirm')!)
    await waitFor(() => expect(onChange).toHaveBeenCalled())
  })

  it('closes dropdown on outside click', async () => {
    renderWithProviders(
      <div>
        <div data-testid="outside">outside</div>
        <ProfileSelector currentProfile={profiles[0]} onProfileChange={noop} />
      </div>,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = document.querySelector('.profile-current')!
    await userEvent.click(header)
    expect(screen.getByText('Dev')).toBeInTheDocument()
    await userEvent.click(screen.getByTestId('outside'))
    expect(screen.queryByText('Dev')).not.toBeInTheDocument()
  })
})
