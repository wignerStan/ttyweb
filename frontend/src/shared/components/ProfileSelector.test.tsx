import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { Profile } from '../../types'
import { ProfileSelector } from './ProfileSelector'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const profiles: Profile[] = [
  { id: 1, profile_key: 'default', name: 'Default', sort_order: 0 },
  { id: 2, profile_key: 'dev', name: 'Dev', sort_order: 1 },
]

const defaultProfile = profiles[0]!
const devProfile = profiles[1]!

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
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
  })

  it('shows Select Profile when no current profile', async () => {
    renderWithProviders(<ProfileSelector currentProfile={null} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Select Profile')).toBeInTheDocument())
  })

  it('opens and closes dropdown', async () => {
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    expect(screen.getByText('Dev')).toBeInTheDocument()
    await userEvent.click(screen.getByText('Dev'))
    expect(screen.queryByText('New Profile')).not.toBeInTheDocument()
  })

  it('lists profiles from API', async () => {
    const onChange = vi.fn()
    renderWithProviders(<ProfileSelector currentProfile={null} onProfileChange={onChange} />)
    await waitFor(() => expect(onChange).toHaveBeenCalled())
    expect(onChange).toHaveBeenCalledWith(defaultProfile)
  })

  it('selects a profile', async () => {
    const onChange = vi.fn()
    renderWithProviders(
      <ProfileSelector currentProfile={defaultProfile} onProfileChange={onChange} />,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('Dev'))
    expect(onChange).toHaveBeenCalledWith(devProfile)
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
    renderWithProviders(
      <ProfileSelector currentProfile={defaultProfile} onProfileChange={onChange} />,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await waitFor(() => expect(screen.getByText('New Profile')).toBeInTheDocument())
    await userEvent.click(screen.getByText('New Profile'))
    const input = screen.getByPlaceholderText('Profile name...')
    await userEvent.type(input, 'New')
    await userEvent.click(document.querySelector('[data-testid="btn-confirm"]')!)
    await waitFor(() => expect(onChange).toHaveBeenCalled())
  })

  it('closes dropdown on outside click', async () => {
    renderWithProviders(
      <div>
        <div data-testid="outside">outside</div>
        <ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />
      </div>,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    expect(screen.getByText('Dev')).toBeInTheDocument()
    await userEvent.click(screen.getByTestId('outside'))
    expect(screen.queryByText('Dev')).not.toBeInTheDocument()
  })

  it('shows active check mark on current profile', async () => {
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    // Default appears in header and dropdown - use getAllByText and find the dropdown item
    const defaultItems = screen.getAllByText('Default')
    const dropdownItem = defaultItems.find((el) => el.closest('.profile-item'))!
    expect(dropdownItem.closest('.profile-item')?.classList.contains('active')).toBe(true)
  })

  it('edits profile name', async () => {
    const fetchMock = mockFetchJSON({ profiles })
    fetchMock.mockResolvedValue({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    const onChange = vi.fn()
    renderWithProviders(
      <ProfileSelector currentProfile={defaultProfile} onProfileChange={onChange} />,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await waitFor(() => expect(screen.getByText('Edit')).toBeInTheDocument())
    await userEvent.click(screen.getByText('Edit'))
    const editInput = screen.getByPlaceholderText('Rename profile...')
    await userEvent.clear(editInput)
    await userEvent.type(editInput, 'Renamed')
    // Click the confirm button in the edit section
    const editSection = document.querySelector('[data-testid="profile-edit-section"]')!
    const confirmBtn = editSection.querySelector('[data-testid="edit-confirm"]')!
    await userEvent.click(confirmBtn)
    await waitFor(() => expect(onChange).toHaveBeenCalled())
    const updated = onChange.mock.calls[0]![0] as Profile
    expect(updated.name).toBe('Renamed')
  })

  it('deletes profile with confirmation', async () => {
    const fetchMock = vi.fn()
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ profiles }) })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', fetchMock)
    const onChange = vi.fn()
    renderWithProviders(<ProfileSelector currentProfile={devProfile} onProfileChange={onChange} />)
    await waitFor(() => expect(screen.getByText('Dev')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await waitFor(() => expect(screen.getByText('Delete')).toBeInTheDocument())
    await userEvent.click(screen.getByText('Delete'))
    await waitFor(() => expect(onChange).toHaveBeenCalled())
  })

  it('does not delete last profile', async () => {
    // Only provide 1 profile so the delete button is disabled
    const singleProfile = [profiles[0]]
    vi.stubGlobal('fetch', mockFetchJSON({ profiles: singleProfile }))
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await waitFor(() => expect(screen.getByText('Delete')).toBeInTheDocument())
    const deleteBtn = screen.getByText('Delete').closest('button')!
    expect(deleteBtn).toBeDisabled()
  })

  it('does not delete when confirm is cancelled', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    const onChange = vi.fn()
    renderWithProviders(<ProfileSelector currentProfile={devProfile} onProfileChange={onChange} />)
    await waitFor(() => expect(screen.getByText('Dev')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('Delete'))
    expect(onChange).not.toHaveBeenCalled()
  })

  it('cancel creation closes the input', async () => {
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('New Profile'))
    await userEvent.click(document.querySelector('[data-testid="btn-cancel"]')!)
    expect(screen.queryByPlaceholderText('Profile name...')).not.toBeInTheDocument()
  })

  it('create profile button is disabled when name is empty', async () => {
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('New Profile'))
    const confirmBtn = document.querySelector(
      '[data-testid="profile-actions"] [data-testid="btn-confirm"]',
    ) as HTMLButtonElement
    expect(confirmBtn).toBeDisabled()
  })

  it('does not create profile with empty name', async () => {
    const onChange = vi.fn()
    renderWithProviders(
      <ProfileSelector currentProfile={defaultProfile} onProfileChange={onChange} />,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('New Profile'))
    const confirmBtn = document.querySelector(
      '[data-testid="profile-actions"] [data-testid="btn-confirm"]',
    ) as HTMLButtonElement
    await userEvent.click(confirmBtn)
    expect(onChange).not.toHaveBeenCalled()
  })

  it('Escape key closes creation input', async () => {
    renderWithProviders(<ProfileSelector currentProfile={defaultProfile} onProfileChange={noop} />)
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('New Profile'))
    const input = screen.getByPlaceholderText('Profile name...')
    await userEvent.type(input, 'Test{Escape}')
    expect(screen.queryByPlaceholderText('Profile name...')).not.toBeInTheDocument()
  })

  it('Enter key submits creation', async () => {
    const fetchMock = mockFetchJSON({ profiles })
    fetchMock.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ profiles }) })
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ id: 3, profile_key: 'test', name: 'Test' }),
    })
    vi.stubGlobal('fetch', fetchMock)
    const onChange = vi.fn()
    renderWithProviders(
      <ProfileSelector currentProfile={defaultProfile} onProfileChange={onChange} />,
    )
    await waitFor(() => expect(screen.getByText('Default')).toBeInTheDocument())
    const header = screen.getByTestId('profile-current')
    await userEvent.click(header)
    await userEvent.click(screen.getByText('New Profile'))
    const input = screen.getByPlaceholderText('Profile name...')
    await userEvent.type(input, 'Test{Enter}')
    await waitFor(() => expect(onChange).toHaveBeenCalled())
  })

  it('handles fetch failure gracefully', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    renderWithProviders(<ProfileSelector currentProfile={null} onProfileChange={noop} />)
    // Should not throw, just show Select Profile
    await waitFor(() => expect(screen.getByText('Select Profile')).toBeInTheDocument())
  })
})
