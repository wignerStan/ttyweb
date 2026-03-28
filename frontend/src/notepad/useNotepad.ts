import { useState, useEffect, useCallback, useRef } from 'react'

export interface Note {
  id: number
  name: string
  content: string
  project_id: string | null
  order_index: number
}

interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
}

const JSON_HEADERS = { 'Content-Type': 'application/json' } as const

async function apiRequest<T>(
  url: string,
  options?: RequestInit,
  defaultError = 'Request failed'
): Promise<ApiResponse<T>> {
  const res = await fetch(url, options)
  const json: ApiResponse<T> = await res.json()
  if (!json.success) {
    throw new Error(json.error ?? defaultError)
  }
  return json
}

export function useNotepad(projectId: string | null = null) {
  const [notes, setNotes] = useState<Note[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const fetchNotes = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const params = projectId ? `?project_id=${encodeURIComponent(projectId)}` : ''
      const json = await apiRequest<Note[]>(`/api/notepad${params}`, undefined, 'Failed to fetch notes')
      setNotes(json.data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect to server')
    } finally {
      setLoading(false)
    }
  }, [projectId])

  useEffect(() => {
    fetchNotes()
  }, [fetchNotes])

  const createNote = useCallback(async (
    name: string,
    content: string = '',
    orderIndex?: number
  ): Promise<Note | null> => {
    try {
      const body: Record<string, unknown> = { name, content }
      if (projectId) body.project_id = projectId
      if (orderIndex !== undefined) body.order_index = orderIndex

      const json = await apiRequest<Note>(
        '/api/notepad',
        { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) },
        'Failed to create note'
      )
      setNotes((prev) => [...prev, json.data])
      return json.data
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create note')
      return null
    }
  }, [projectId])

  const updateNote = useCallback(
    (id: number, updates: { name?: string; content?: string }, debounceMs: number = 0) => {
      const doUpdate = async () => {
        try {
          await apiRequest<Note>(
            `/api/notepad/${id}`,
            { method: 'PUT', headers: JSON_HEADERS, body: JSON.stringify(updates) },
            'Failed to update note'
          )
          setNotes((prev) =>
            prev.map((n) => (n.id === id ? { ...n, ...updates } : n))
          )
        } catch (err) {
          setError(err instanceof Error ? err.message : 'Failed to update note')
        }
      }

      if (debounceRef.current) clearTimeout(debounceRef.current)
      if (debounceMs > 0) {
        debounceRef.current = setTimeout(doUpdate, debounceMs)
      } else {
        void doUpdate()
      }
    },
    []
  )

  const deleteNote = useCallback(async (id: number) => {
    try {
      await apiRequest<null>(
        `/api/notepad/${id}`,
        { method: 'DELETE' },
        'Failed to delete note'
      )
      setNotes((prev) => prev.filter((n) => n.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete note')
    }
  }, [])

  const reorderNotes = useCallback(async (reorder: Array<{ id: number; order_index: number }>) => {
    try {
      await apiRequest<null>(
        '/api/notepad/reorder',
        { method: 'PATCH', headers: JSON_HEADERS, body: JSON.stringify(reorder) },
        'Failed to reorder notes'
      )
      const indexMap = new Map(reorder.map((r) => [r.id, r.order_index]))
      setNotes((prev) =>
        prev
          .map((n) => ({ ...n, order_index: indexMap.get(n.id) ?? n.order_index }))
          .sort((a, b) => a.order_index - b.order_index)
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reorder notes')
    }
  }, [])

  return {
    notes,
    loading,
    error,
    fetchNotes,
    createNote,
    updateNote,
    deleteNote,
    reorderNotes,
  }
}
