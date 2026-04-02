import { useCallback, useEffect, useState } from 'react'
import { getAuthHeaders } from '../utils/auth'
import type { AISession, ApiResponse, ConversationMessage } from './types'

export function useConversations(projectPath: string | null) {
  const [sessions, setSessions] = useState<AISession[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const fetchSessions = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const params = new URLSearchParams()
      if (projectPath) params.set('project', projectPath)
      const qs = params.toString()
      const url = `/api/ai/sessions${qs ? `?${qs}` : ''}`
      const res = await fetch(url, { headers: getAuthHeaders() })
      const json: ApiResponse<AISession[]> = await res.json()
      if (json.success) {
        setSessions(json.data)
      } else {
        setError(json.error ?? 'Failed to fetch sessions')
      }
    } catch {
      setError('Failed to connect to server')
    } finally {
      setLoading(false)
    }
  }, [projectPath])

  useEffect(() => {
    fetchSessions()
  }, [fetchSessions])

  return { sessions, loading, error, refetch: fetchSessions }
}

async function fetchMessages(
  sessionId: string,
  endpoint: 'conversation' | 'refresh',
): Promise<{ data: ConversationMessage[]; error: string }> {
  const res = await fetch(`/api/ai/sessions/${encodeURIComponent(sessionId)}/${endpoint}`, {
    headers: getAuthHeaders(),
  })
  const json: ApiResponse<ConversationMessage[]> = await res.json()
  if (json.success) {
    return { data: json.data, error: '' }
  }
  return { data: [], error: json.error ?? `Failed to ${endpoint} conversation` }
}

export function useConversation(sessionId: string | null) {
  const [messages, setMessages] = useState<ConversationMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(
    async (endpoint: 'conversation' | 'refresh') => {
      if (!sessionId) return
      setLoading(true)
      setError('')
      try {
        const result = await fetchMessages(sessionId, endpoint)
        setMessages(result.data)
        setError(result.error)
      } catch {
        setError('Failed to connect to server')
      } finally {
        setLoading(false)
      }
    },
    [sessionId],
  )

  useEffect(() => {
    setMessages([])
    setError('')
    load('conversation')
  }, [load])

  return {
    messages,
    loading,
    error,
    refetch: () => load('conversation'),
    refresh: () => load('refresh'),
  }
}
