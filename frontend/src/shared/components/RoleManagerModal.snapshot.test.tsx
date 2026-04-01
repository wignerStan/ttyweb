import { describe, expect, it, vi } from 'vitest'
import { mockFetchSuccess } from '../../test-helpers'
import { renderWithProviders } from '../../test-utils'
import type { AiRole } from '../../types'
import { RoleManagerModal } from './RoleManagerModal'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const mockRoles: AiRole[] = [
  { id: 'custom-1', emoji: '🔬', label: 'Researcher', desc: 'Deep research', isCustom: true },
  { id: 'custom-2', emoji: '📝', label: 'Writer', desc: 'Technical writing', isCustom: true },
]

vi.stubGlobal('fetch', mockFetchSuccess({}))

describe('RoleManagerModal snapshot', () => {
  it('renders role manager with custom roles', () => {
    const { container } = renderWithProviders(
      <RoleManagerModal
        open={true}
        onClose={() => {}}
        roles={mockRoles}
        onRolesChanged={() => {}}
      />,
    )
    expect(container).toMatchSnapshot()
  })
})
