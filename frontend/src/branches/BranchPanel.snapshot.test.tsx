import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { BranchPanel } from './BranchPanel'
import type { BranchInfo } from './useBranches'

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

vi.mock('./useBranches', () => ({
  useBranches: (_repoPath: string | null) => ({
    branches: mockBranches,
    loading: false,
    error: null,
    refetch: () => Promise.resolve(),
    createBranch: () => Promise.resolve(),
    deleteBranch: () => Promise.resolve(),
  }),
}))

describe('BranchPanel snapshot', () => {
  it('renders branch list with current, ahead/behind badges', () => {
    const { container } = renderWithProviders(<BranchPanel repoPath="/home/user/repo" />)
    expect(container).toMatchSnapshot()
  })
})
