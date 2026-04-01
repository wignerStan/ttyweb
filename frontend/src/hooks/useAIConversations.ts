import { useCallback, useEffect, useRef, useState } from 'react'
import type { AiConversation } from '../types'
import { getAuthHeader, getAuthHeaders } from '../utils/auth'

export function useAIConversations(paneKey: string | null) {
  const [conversations, setConversations] = useState<AiConversation[]>([])
  const [loading, setLoading] = useState(false)
  const eventSourceRef = useRef<EventSource | null>(null)

  const fetchConversations = useCallback(async () => {
    if (!paneKey) return
    setLoading(true)
    try {
      const res = await fetch(`/api/tasks/events/${encodeURIComponent(paneKey)}`, {
        headers: getAuthHeaders(),
      })
      const data = await res.json()
      setConversations(data.conversations || [])
    } catch (_err) {
    } finally {
      setLoading(false)
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
