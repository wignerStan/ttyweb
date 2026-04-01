import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { BranchPanel } from './BranchPanel'
import type { BranchInfo } from './useBranches'

const mockRefetch = vi.fn()
const mockCreateBranch = vi.fn()
const mockDeleteBranch = vi.fn()

const mockBranches: BranchInfo[] = [
  {
    name: 'main',
    is_default: true,
    is_remote: false,
    is_current: true,
    head_hash: 'abc1234',
    ahead: 0,
    behind: 0,
  },
  {
    name: 'feature/login',
    is_default: false,
    is_remote: false,
    is_current: false,
    head_hash: 'def5678',
    ahead: 3,
    behind: 1,
  },
  {
    name: 'fix/header-bug',
    is_default: false,
    is_remote: false,
    is_current: false,
    head_hash: 'ghi9012',
    ahead: 0,
    behind: 5,
  },
]

const defaultHookState = {
  branches: mockBranches,
  loading: false,
  error: null as string | null,
  refetch: mockRefetch,
  createBranch: mockCreateBranch,
  deleteBranch: mockDeleteBranch,
}

let hookState = { ...defaultHookState, branches: [...mockBranches] }

vi.mock('./useBranches', () => ({
  useBranches: (_repoPath: string | null) => ({ ...hookState }),
}))

describe('BranchPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    hookState = { ...defaultHookState, branches: [...mockBranches] }
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders branch list', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())
    expect(screen.getByText('feature/login')).toBeInTheDocument()
    expect(screen.getByText('fix/header-bug')).toBeInTheDocument()
  })

  it('highlights current branch with "current" badge', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('current')).toBeInTheDocument())
    const mainEl = screen.getByTitle('abc1234')
    expect(mainEl.textContent).toBe('main')
  })

  it('shows default branch badge on current branch', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => {
      const defaults = screen.getAllByText('default')
      expect(defaults.length).toBe(1)
    })
  })

  it('shows ahead/behind status badges', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('\u21913')).toBeInTheDocument()) // ↑3
    expect(screen.getByText('\u21931')).toBeInTheDocument() // ↓1
    expect(screen.getByText('\u21935')).toBeInTheDocument() // ↓5
  })

  it('does not show status badges when ahead and behind are both 0', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())
    const mainContainer = screen.getByTitle('abc1234').closest('.branch-item--current')
    const statusBadges = mainContainer?.querySelectorAll('.badge-success, .badge-warning')
    expect(statusBadges?.length).toBe(0)
  })

  it('shows empty state when no branches', async () => {
    hookState = { ...defaultHookState, branches: [] }
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('No branches found')).toBeInTheDocument())
  })

  it('shows no repository path message when repoPath is null', () => {
    renderWithProviders(<BranchPanel repoPath={null} />)
    expect(screen.getByText('No repository path specified')).toBeInTheDocument()
  })

  it('shows error state', async () => {
    hookState = { ...defaultHookState, branches: [], error: 'Connection refused' }
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() =>
      expect(screen.getByText(/Failed to load branches: Connection refused/)).toBeInTheDocument(),
    )
  })

  it('create branch flow: opens input, types name, submits', async () => {
    mockCreateBranch.mockResolvedValue(undefined)
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Create branch'))
    expect(screen.getByPlaceholderText('Branch name...')).toBeInTheDocument()

    const input = screen.getByPlaceholderText('Branch name...')
    await userEvent.type(input, 'new-feature-branch')

    await userEvent.click(screen.getByTitle('Create'))
    await waitFor(() => expect(mockCreateBranch).toHaveBeenCalledWith('new-feature-branch'))
  })

  it('create branch: Enter key submits', async () => {
    mockCreateBranch.mockResolvedValue(undefined)
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Create branch'))
    const input = screen.getByPlaceholderText('Branch name...')
    await userEvent.type(input, 'enter-branch{Enter}')
    await waitFor(() => expect(mockCreateBranch).toHaveBeenCalledWith('enter-branch'))
  })

  it('create branch: Escape key cancels', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Create branch'))
    expect(screen.getByPlaceholderText('Branch name...')).toBeInTheDocument()

    const input = screen.getByPlaceholderText('Branch name...')
    await userEvent.type(input, 'partial{Escape}')
    expect(screen.queryByPlaceholderText('Branch name...')).not.toBeInTheDocument()
  })

  it('create branch: cancel button closes input', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Create branch'))
    await userEvent.click(screen.getByTitle('Cancel'))
    expect(screen.queryByPlaceholderText('Branch name...')).not.toBeInTheDocument()
  })

  it('create branch: does not submit with empty name', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Create branch'))
    const createBtn = screen.getByTitle('Create') as HTMLButtonElement
    expect(createBtn.disabled).toBe(true)
    await userEvent.click(createBtn)
    expect(mockCreateBranch).not.toHaveBeenCalled()
  })

  it('create branch: shows error on failure', async () => {
    mockCreateBranch.mockRejectedValue(new Error('branch already exists'))
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Create branch'))
    const input = screen.getByPlaceholderText('Branch name...')
    await userEvent.type(input, 'duplicate-branch')
    await userEvent.click(screen.getByTitle('Create'))

    await waitFor(() => expect(screen.getByText('branch already exists')).toBeInTheDocument())
  })

  it('delete branch flow: confirms and calls deleteBranch', async () => {
    mockDeleteBranch.mockResolvedValue(undefined)
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('feature/login')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Delete feature/login'))
    // ConfirmDialog should appear — click the confirm (Delete) button inside the dialog
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /^Delete$/ }))
    await waitFor(() => expect(mockDeleteBranch).toHaveBeenCalledWith('feature/login'))
  })

  it('delete branch: does not delete when confirm is cancelled', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('feature/login')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Delete feature/login'))
    // ConfirmDialog should appear — click cancel
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /cancel/i }))
    expect(mockDeleteBranch).not.toHaveBeenCalled()
  })

  it('delete branch: shows error on failure', async () => {
    mockDeleteBranch.mockRejectedValue(new Error('not fully merged'))
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('feature/login')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Delete feature/login'))
    // ConfirmDialog should appear — click the confirm (Delete) button inside the dialog
    const dialog = screen.getByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: /^Delete$/ }))
    await waitFor(() => expect(screen.getByText('not fully merged')).toBeInTheDocument())
  })

  it('refresh button calls refetch', async () => {
    renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    await waitFor(() => expect(screen.getByText('main')).toBeInTheDocument())

    await userEvent.click(screen.getByTitle('Refresh'))
    expect(mockRefetch).toHaveBeenCalledTimes(1)
  })

  it('does not show refresh button when repoPath is null', () => {
    renderWithProviders(<BranchPanel repoPath={null} />)
    expect(screen.queryByTitle('Refresh')).not.toBeInTheDocument()
  })
})
