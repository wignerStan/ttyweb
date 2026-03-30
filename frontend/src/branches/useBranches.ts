import { useCallback, useEffect, useState } from 'react'
import { getAuthHeader } from '../utils/auth'

export interface BranchInfo {
  name: string
  is_default: boolean
  is_remote: boolean
  is_current: boolean
  head_hash: string
  ahead: number
  behind: number
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
      const authHeader = getAuthHeader()
      const res = await fetch(`/api/branches?repo=${encodeURIComponent(repoPath)}`, {
        headers: authHeader ? { Authorization: authHeader } : undefined,
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || `HTTP ${res.status}`)
      }
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
      const authHeader = getAuthHeader()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (authHeader) headers.Authorization = authHeader
      const res = await fetch(`/api/branches?repo=${encodeURIComponent(repoPath)}`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ name }),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || `HTTP ${res.status}`)
      }
      await fetchBranches()
    },
    [repoPath, fetchBranches],
  )

  const deleteBranch = useCallback(
    async (name: string) => {
      if (!repoPath) return
      const authHeader = getAuthHeader()
      const res = await fetch(
        `/api/branches/${encodeURIComponent(name)}?repo=${encodeURIComponent(repoPath)}`,
        {
          method: 'DELETE',
          headers: authHeader ? { Authorization: authHeader } : undefined,
        },
      )
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || `HTTP ${res.status}`)
      }
      await fetchBranches()
    },
    [repoPath, fetchBranches],
  )

  return { branches, loading, error, refetch: fetchBranches, createBranch, deleteBranch }
}
