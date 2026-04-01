import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test-utils'
import type { Profile } from '../../types'
import { ProfileSelector } from './ProfileSelector'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const mockProfiles: Profile[] = [
  { id: 1, profile_key: 'default', name: 'Default', sort_order: 0 },
  { id: 2, profile_key: 'dev', name: 'Dev', sort_order: 1 },
]

vi.stubGlobal(
  'fetch',
  vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ profiles: mockProfiles }) }),
)

describe('ProfileSelector snapshot', () => {
  it('renders profile selector with current profile', () => {
    const { container } = renderWithProviders(
      <ProfileSelector currentProfile={mockProfiles[0]!} onProfileChange={() => {}} />,
    )
    expect(container).toMatchSnapshot()
  })
})
