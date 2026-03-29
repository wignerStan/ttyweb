import { useCallback, useEffect, useState } from 'react'
import { getAuthHeader } from '../../../../utils/auth'
import { BUTLER_API_BASE, POLL_ACTIVITY_MS } from '../constants'
import type { ActivityEvent } from '../types'

function toError(e: unknown): Error {
  return e instanceof Error ? e : new Error(String(e))
}

export function useActivityEvents(studyId?: string, limit = 20) {
  const [events, setEvents] = useState<ActivityEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const refetch = useCallback(async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({ limit: String(limit) })
      if (studyId) params.set('study_id', studyId)
      const authHeader = getAuthHeader()
      const res = await fetch(`${BUTLER_API_BASE}/activity_events?${params}`, {
        headers: authHeader ? { Authorization: authHeader } : undefined,
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json = await res.json()
      const data: ActivityEvent[] = json?.data?.activity_events ?? []
      setEvents(data)
      setError(null)
    } catch (e: unknown) {
      setError(toError(e))
    } finally {
      setLoading(false)
    }
  }, [studyId, limit])

  useEffect(() => {
    refetch()
    const interval = setInterval(refetch, POLL_ACTIVITY_MS)
    return () => clearInterval(interval)
  }, [refetch])

  return { events, loading, error, refetch }
}
