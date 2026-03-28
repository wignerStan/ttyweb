import { useState, useEffect, useCallback } from 'react'
import type { AISession, ConversationMessage, ApiResponse } from './types'
import { getAuthHeader } from '../utils/auth'

function authHeaders(): Record<string, string> {
  const auth = getAuthHeader()
  return auth ? { Authorization: auth } : {}
}

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
      const url = `/api/ai/sessions${params.toString() ? `?${params.toString()}` : ''}`
      const res = await fetch(url, { headers: authHeaders() })
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

export function useConversation(sessionId: string | null) {
  const [messages, setMessages] = useState<ConversationMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [session, setSession] = useState<AISession | null>(null)

  const fetchConversation = useCallback(async () => {
    if (!sessionId) return
    setLoading(true)
    setError('')
    try {
      const res = await fetch(
        `/api/ai/sessions/${encodeURIComponent(sessionId)}/conversation`,
        { headers: authHeaders() }
      )
      const json: ApiResponse<ConversationMessage[]> = await res.json()
      if (json.success) {
        setMessages(json.data)
      } else {
        setError(json.error ?? 'Failed to fetch conversation')
      }
    } catch {
      setError('Failed to connect to server')
    } finally {
      setLoading(false)
    }
  }, [sessionId])

  const refreshConversation = useCallback(async () => {
    if (!sessionId) return
    setLoading(true)
    setError('')
    try {
      const res = await fetch(
        `/api/ai/sessions/${encodeURIComponent(sessionId)}/refresh`,
        { headers: authHeaders() }
      )
      const json: ApiResponse<ConversationMessage[]> = await res.json()
      if (json.success) {
        setMessages(json.data)
      } else {
        setError(json.error ?? 'Failed to refresh conversation')
      }
    } catch {
      setError('Failed to connect to server')
    } finally {
      setLoading(false)
    }
  }, [sessionId])

  const setSessionInfo = useCallback((s: AISession | null) => {
    setSession(s)
  }, [])

  useEffect(() => {
    setMessages([])
    setError('')
    fetchConversation()
  }, [fetchConversation])

  return {
    messages,
    loading,
    error,
    session,
    setSessionInfo,
    refetch: fetchConversation,
    refresh: refreshConversation,
  }
}
