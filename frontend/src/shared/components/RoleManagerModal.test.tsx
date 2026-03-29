import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { mockFetchSuccess } from '../../test-helpers'
import { renderWithProviders } from '../../test-utils'
import type { AiRole } from '../../types'
import { RoleManagerModal } from './RoleManagerModal'

vi.mock('../../utils/auth', () => ({ getAuthHeader: () => 'Bearer test-token' }))

const customRoles: AiRole[] = [
  { id: 'custom-1', emoji: '🔬', label: 'Researcher', desc: 'Deep research', isCustom: true },
  { id: 'custom-2', emoji: '📝', label: 'Writer', desc: 'Technical writing', isCustom: true },
]

describe('RoleManagerModal', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetchSuccess({}))
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('does not render when closed', () => {
    renderWithProviders(
      <RoleManagerModal open={false} onClose={() => {}} roles={[]} onRolesChanged={() => {}} />,
    )
    expect(screen.queryByText('自定义角色')).not.toBeInTheDocument()
  })

  it('renders role list when open', () => {
    renderWithProviders(
      <RoleManagerModal
        open={true}
        onClose={() => {}}
        roles={customRoles}
        onRolesChanged={() => {}}
      />,
    )
    expect(screen.getByText('Researcher')).toBeInTheDocument()
    expect(screen.getByText('Writer')).toBeInTheDocument()
  })

  it('shows empty state for no custom roles', () => {
    renderWithProviders(
      <RoleManagerModal open={true} onClose={() => {}} roles={[]} onRolesChanged={() => {}} />,
    )
    expect(screen.getByText('暂无自定义角色')).toBeInTheDocument()
  })

  it('closes on overlay click', async () => {
    const onClose = vi.fn()
    renderWithProviders(
      <RoleManagerModal open={true} onClose={onClose} roles={[]} onRolesChanged={() => {}} />,
    )
    await userEvent.click(screen.getByText('暂无自定义角色').closest('.role-modal-overlay')!)
    expect(onClose).toHaveBeenCalled()
  })

  it('creates a new role', async () => {
    const postMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', postMock)
    const onRolesChanged = vi.fn()
    renderWithProviders(
      <RoleManagerModal
        open={true}
        onClose={() => {}}
        roles={[]}
        onRolesChanged={onRolesChanged}
      />,
    )

    await userEvent.click(screen.getByText('新建角色'))
    await userEvent.type(screen.getByPlaceholderText('ID (英文)'), 'tester')
    await userEvent.type(screen.getByPlaceholderText('名称'), 'Tester')
    await userEvent.type(screen.getByPlaceholderText('描述'), 'Testing role')
    await userEvent.click(screen.getByText('保存'))

    await waitFor(() =>
      expect(postMock).toHaveBeenCalledWith(
        '/api/roles',
        expect.objectContaining({ method: 'POST' }),
      ),
    )
  })

  it('edits an existing role', async () => {
    const putMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', putMock)
    renderWithProviders(
      <RoleManagerModal
        open={true}
        onClose={() => {}}
        roles={customRoles}
        onRolesChanged={() => {}}
      />,
    )

    const editButtons = screen.getAllByTitle('编辑')
    await userEvent.click(editButtons[0])
    await userEvent.type(screen.getByPlaceholderText('名称'), ' Updated')
    await userEvent.click(screen.getByText('保存'))

    await waitFor(() => expect(putMock).toHaveBeenCalled())
  })

  it('deletes a role', async () => {
    const delMock = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) })
    vi.stubGlobal('fetch', delMock)
    const onRolesChanged = vi.fn()
    renderWithProviders(
      <RoleManagerModal
        open={true}
        onClose={() => {}}
        roles={customRoles}
        onRolesChanged={onRolesChanged}
      />,
    )

    const deleteButtons = screen.getAllByTitle('删除')
    await userEvent.click(deleteButtons[0])

    await waitFor(() =>
      expect(delMock).toHaveBeenCalledWith(
        '/api/roles/custom-1',
        expect.objectContaining({ method: 'DELETE' }),
      ),
    )
  })
})
