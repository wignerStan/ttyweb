import { useCallback, useEffect, useRef, useState } from 'react'
import { getAuthHeader } from '../../../../utils/auth'
import { BUTLER_API_BASE } from '../constants'
import type { TaskEventDetail, TaskRunDetail } from '../types'

const POLL_MS = 3_000
const TERMINAL_STATES = new Set(['succeeded', 'failed', 'cancelled'])

export function useRunDetail(runId: string | null) {
  const [run, setRun] = useState<TaskRunDetail | null>(null)
  const [events, setEvents] = useState<TaskEventDetail[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  const fetchDetail = useCallback(async (id: string, signal: AbortSignal) => {
    try {
      const authHeader = getAuthHeader()
      const headers: Record<string, string> = {}
      if (authHeader) headers.Authorization = authHeader
      const [runRes, eventsRes] = await Promise.all([
        fetch(`${BUTLER_API_BASE}/runs/${id}`, { signal, headers }),
        fetch(`${BUTLER_API_BASE}/runs/${id}/events`, { signal, headers }),
      ])
      if (signal.aborted) return null

      if (!runRes.ok) throw new Error(`Run: HTTP ${runRes.status}`)
      if (!eventsRes.ok) throw new Error(`Events: HTTP ${eventsRes.status}`)

      const runJson = await runRes.json()
      const eventsJson = await eventsRes.json()

      const runData: TaskRunDetail = runJson?.data ?? runJson
      const eventsData: TaskEventDetail[] = eventsJson?.data?.events ?? []

      setRun(runData)
      setEvents(eventsData)
      setError(null)
      return runData
    } catch (e: unknown) {
      if (!signal.aborted) {
        const msg = e instanceof Error ? e.message : 'Failed to fetch run detail'
        setError(msg)
      }
      return null
    }
  }, [])

  useEffect(() => {
    if (!runId) {
      setRun(null)
      setEvents([])
      setError(null)
      setLoading(false)
      return
    }

    const controller = new AbortController()
    abortRef.current = controller
    let timer: ReturnType<typeof setTimeout> | null = null

    const poll = async () => {
      const result = await fetchDetail(runId, controller.signal)
      if (controller.signal.aborted) return
      if (result && !TERMINAL_STATES.has(result.state)) {
        timer = setTimeout(poll, POLL_MS)
      }
    }

    setLoading(true)
    poll().finally(() => {
      if (!controller.signal.aborted) setLoading(false)
    })

    return () => {
      controller.abort()
      if (timer) clearTimeout(timer)
    }
  }, [runId, fetchDetail])

  return { run, events, loading, error }
}
