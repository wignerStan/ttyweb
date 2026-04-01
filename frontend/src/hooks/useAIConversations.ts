import { useCallback, useEffect, useRef, useState } from 'react'
import type { AiConversation } from '../types'
import { getAuthHeader } from '../utils/auth'

export function useAIConversations(paneKey: string | null) {
  const [conversations, setConversations] = useState<AiConversation[]>([])
  const [loading, setLoading] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)
  const fetchInProgressRef = useRef(false)

  const fetchConversations = useCallback(async () => {
    if (!paneKey) return
    if (fetchInProgressRef.current) return
    fetchInProgressRef.current = true
    setLoading(true)
    try {
      const authHeader = getAuthHeader()
      const res = await fetch(`/api/tasks/events/${encodeURIComponent(paneKey)}`, {
        headers: authHeader ? { Authorization: authHeader } : undefined,
      })
      const data = await res.json()
      setConversations(data.conversations || [])
    } catch (_err) {
      // Silently handle — the SSE connection will retry
    } finally {
      setLoading(false)
      fetchInProgressRef.current = false
    }
  }, [paneKey])

  useEffect(() => {
    if (!paneKey) {
      setConversations([])
      return
    }

    fetchConversations()

    const auth = getAuthHeader() || ''
    const es = new EventSource(
      `/api/tasks/events/stream/${encodeURIComponent(paneKey)}?auth=${encodeURIComponent(auth)}`,
    )
    eventSourceRef.current = es

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (['task_started', 'task_completed', 'task_failed', 'task_waiting'].includes(data.type)) {
          fetchConversations()
        }
      } catch (_err) {}
    }

    es.onerror = () => {}

    return () => {
      es.close()
      eventSourceRef.current = null
    }
  }, [paneKey, fetchConversations])

  return { conversations, loading, refetch: fetchConversations }
}
