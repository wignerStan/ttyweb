import { Check, GitBranch, Loader2, Plus, Trash2, X } from 'lucide-react'
import { useCallback, useRef, useState } from 'react'
import { useBranches } from './useBranches'

interface BranchPanelProps {
  repoPath: string | null
}

export function BranchPanel({ repoPath }: BranchPanelProps) {
  const { branches, loading, error, refetch, createBranch, deleteBranch } = useBranches(repoPath)
  const [isCreating, setIsCreating] = useState(false)
  const [newBranchName, setNewBranchName] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [deletingBranch, setDeletingBranch] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleCreate = useCallback(async () => {
    const name = newBranchName.trim()
    if (!name || busy) return
    setBusy(true)
    setActionError(null)
    try {
      await createBranch(name)
      setNewBranchName('')
      setIsCreating(false)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to create branch'
      setActionError(msg)
    } finally {
      setBusy(false)
    }
  }, [newBranchName, busy, createBranch])

  const handleDelete = useCallback(
    async (name: string) => {
      if (busy) return
      if (!confirm(`Delete branch "${name}"?`)) return
      setBusy(true)
      setDeletingBranch(name)
      setActionError(null)
      try {
        await deleteBranch(name)
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : 'Failed to delete branch'
        setActionError(msg)
      } finally {
        setBusy(false)
        setDeletingBranch(null)
      }
    },
    [busy, deleteBranch],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') handleCreate()
      if (e.key === 'Escape') {
        setIsCreating(false)
        setNewBranchName('')
      }
    },
    [handleCreate],
  )

  const currentBranch = branches.find((b) => b.is_current)
  const otherBranches = branches.filter((b) => !b.is_current)

  return (
    <div className="branch-panel">
      <div className="branch-header">
        <span className="branch-title">
          <GitBranch size={14} />
          Branches
        </span>
        <div className="branch-header-actions">
          {repoPath && (
            <button
              type="button"
              className="btn-icon"
              onClick={refetch}
              title="Refresh"
              disabled={loading}
            >
              {loading ? <Loader2 size={12} className="spinning" /> : null}
            </button>
          )}
          <button
            type="button"
            className="branch-add-btn"
            onClick={() => setIsCreating(true)}
            title="Create branch"
          >
            <Plus size={14} />
          </button>
        </div>
      </div>

      {error && <div className="branch-error">Failed to load branches: {error}</div>}
      {actionError && <div className="branch-error">{actionError}</div>}

      {!repoPath && <div className="branch-empty">No repository path specified</div>}

      {isCreating && (
        <div className="branch-create-row">
          <input
            ref={inputRef}
            type="text"
            value={newBranchName}
            onChange={(e) => setNewBranchName(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Branch name..."
            className="branch-input"
            disabled={busy}
          />
          <button
            type="button"
            onClick={handleCreate}
            disabled={busy || !newBranchName.trim()}
            className="btn-sm btn-confirm"
            title="Create"
          >
            <Check size={12} />
          </button>
          <button
            type="button"
            onClick={() => {
              setIsCreating(false)
              setNewBranchName('')
            }}
            className="btn-sm btn-cancel"
            title="Cancel"
          >
            <X size={12} />
          </button>
        </div>
      )}

      <div className="branch-list">
        {loading && branches.length === 0 && (
          <div className="branch-loading">
            <Loader2 size={14} className="spinning" />
            <span>Loading branches...</span>
          </div>
        )}

        {!loading && repoPath && branches.length === 0 && (
          <div className="branch-empty">No branches found</div>
        )}

        {currentBranch && (
          <div className="branch-item branch-item--current">
            <div className="branch-row">
              <span className="branch-name branch-name--current" title={currentBranch.head_hash}>
                {currentBranch.name}
              </span>
              <span className="branch-current-badge">current</span>
              <BranchStatus ahead={currentBranch.ahead} behind={currentBranch.behind} />
              <div className="branch-actions">
                <span className="branch-default-badge" title="Default branch">
                  {currentBranch.is_default ? 'default' : ''}
                </span>
              </div>
            </div>
          </div>
        )}

        {otherBranches.map((branch) => (
          <div key={branch.name} className="branch-item">
            <div className="branch-row">
              <span className="branch-name" title={branch.head_hash}>
                {branch.name}
              </span>
              <BranchStatus ahead={branch.ahead} behind={branch.behind} />
              <div className="branch-actions">
                {branch.is_default && (
                  <span className="branch-default-badge" title="Default branch">
                    default
                  </span>
                )}
                <button
                  type="button"
                  className="btn-icon btn-danger"
                  onClick={() => handleDelete(branch.name)}
                  title={`Delete ${branch.name}`}
                  disabled={deletingBranch === branch.name}
                >
                  {deletingBranch === branch.name ? (
                    <Loader2 size={12} className="spinning" />
                  ) : (
                    <Trash2 size={12} />
                  )}
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function BranchStatus({ ahead, behind }: { ahead: number; behind: number }) {
  if (ahead === 0 && behind === 0) return null

  return (
    <span className="branch-status" title={`${ahead} ahead, ${behind} behind`}>
      {ahead > 0 && <span className="branch-status-ahead">&#8593;{ahead}</span>}
      {behind > 0 && <span className="branch-status-behind">&#8595;{behind}</span>}
    </span>
  )
}
