import { useCallback, useEffect, useState } from 'react'
import { getAuthHeaders } from '../utils/auth'

export interface BranchInfo {
  name: string
  is_default: boolean
  is_remote: boolean
  is_current: boolean
  head_hash: string
  ahead: number
  behind: number
}

async function assertOk(res: Response): Promise<void> {
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error || `HTTP ${res.status}`)
  }
}

export function useBranches(repoPath: string | null) {
  const [branches, setBranches] = useState<BranchInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchBranches = useCallback(async () => {
    if (!repoPath) return
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/branches?repo=${encodeURIComponent(repoPath)}`, {
        headers: getAuthHeaders(),
      })
      await assertOk(res)
      const json = await res.json()
      setBranches(json.data || [])
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to fetch branches'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }, [repoPath])

  useEffect(() => {
    fetchBranches()
  }, [fetchBranches])

  const createBranch = useCallback(
    async (name: string) => {
      if (!repoPath) return
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }
      const res = await fetch(`/api/branches?repo=${encodeURIComponent(repoPath)}`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ name }),
      })
      await assertOk(res)
      await fetchBranches()
    },
    [repoPath, fetchBranches],
  )

  const deleteBranch = useCallback(
    async (name: string) => {
      if (!repoPath) return
      const res = await fetch(
        `/api/branches/${encodeURIComponent(name)}?repo=${encodeURIComponent(repoPath)}`,
        {
          method: 'DELETE',
          headers: getAuthHeaders(),
        },
      )
      await assertOk(res)
      await fetchBranches()
    },
    [repoPath, fetchBranches],
  )

  return { branches, loading, error, refetch: fetchBranches, createBranch, deleteBranch }
}
