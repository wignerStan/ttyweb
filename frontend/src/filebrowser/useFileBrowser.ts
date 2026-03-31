import { useCallback, useState } from 'react'
import { getAuthHeader } from '../utils/auth'

export interface FSEntry {
  name: string
  is_dir: boolean
  size: number
  mod_time: string
}

interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
}

export interface UseFileBrowserReturn {
  entries: FSEntry[]
  loading: boolean
  error: string | null
  navigateTo: (dir: string) => Promise<void>
}

async function fetchEntries(path: string): Promise<FSEntry[]> {
  const params = new URLSearchParams({ path })
  const auth = getAuthHeader()
  const headers: Record<string, string> | undefined = auth ? { Authorization: auth } : undefined

  const res = await fetch(`/api/fs?${params.toString()}`, { headers })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    throw new Error(body?.error ?? `HTTP ${res.status}`)
  }

  const json: ApiResponse<FSEntry[]> = await res.json()
  if (!json.success) {
    throw new Error(json.error ?? 'Failed to list directory')
  }

  return json.data
}

export function useFileBrowser(): UseFileBrowserReturn {
  const [entries, setEntries] = useState<FSEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const navigateTo = useCallback(async (dir: string) => {
    setLoading(true)
    setError(null)
    try {
      const result = await fetchEntries(dir)
      setEntries(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to list directory')
    } finally {
      setLoading(false)
    }
  }, [])

  return {
    entries,
    loading,
    error,
    navigateTo,
  }
}
