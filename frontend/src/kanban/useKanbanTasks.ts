import { useCallback, useEffect, useState } from 'react'
import type { KanbanApiResponse, KanbanComment, KanbanTask } from './types'

interface UseKanbanTasksReturn {
  tasks: KanbanTask[]
  loading: boolean
  error: string | null
  fetchTasks: () => Promise<void>
  createTask: (fields: Partial<TaskFields>) => Promise<KanbanTask | null>
  updateTask: (id: string, fields: Partial<TaskFields>) => Promise<KanbanTask | null>
  moveTask: (id: string, status: string, orderIndex: number) => Promise<KanbanTask | null>
  deleteTask: (id: string) => Promise<boolean>
  fetchComments: (taskId: string) => Promise<KanbanComment[]>
  createComment: (taskId: string, content: string) => Promise<KanbanComment | null>
}

export interface TaskFields {
  title: string
  description?: string
  status?: string
  priority?: number
  tags?: string[]
  due_date?: string | null
}

async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`API ${res.status}: ${body || res.statusText}`)
  }
  return res.json() as Promise<T>
}

export function useKanbanTasks(): UseKanbanTasksReturn {
  const [tasks, setTasks] = useState<KanbanTask[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchTasks = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const statuses = ['todo', 'in_progress', 'done', 'archived'] as const
      const results = await Promise.all(
        statuses.map(async (status) => {
          try {
            const res = await apiRequest<KanbanApiResponse<KanbanTask[]>>(
              `/api/kanban/tasks?status=${status}`,
            )
            return res.data
          } catch {
            return [] as KanbanTask[]
          }
        }),
      )
      setTasks(results.flat())
    } finally {
      setLoading(false)
    }
  }, [])

  const createTask = useCallback(
    async (fields: Partial<TaskFields>): Promise<KanbanTask | null> => {
      try {
        const res = await apiRequest<KanbanApiResponse<KanbanTask>>('/api/kanban/tasks', {
          method: 'POST',
          body: JSON.stringify(fields),
        })
        const task = res.data
        setTasks((prev) => [...prev, task])
        return task
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to create task')
        return null
      }
    },
    [],
  )

  const updateTask = useCallback(
    async (id: string, fields: Partial<TaskFields>): Promise<KanbanTask | null> => {
      try {
        const res = await apiRequest<KanbanApiResponse<KanbanTask>>(`/api/kanban/tasks/${id}`, {
          method: 'PUT',
          body: JSON.stringify(fields),
        })
        const updated = res.data
        setTasks((prev) => prev.map((t) => (t.id === id ? updated : t)))
        return updated
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update task')
        return null
      }
    },
    [],
  )

  const moveTask = useCallback(
    async (id: string, status: string, orderIndex: number): Promise<KanbanTask | null> => {
      try {
        const res = await apiRequest<KanbanApiResponse<KanbanTask>>(
          `/api/kanban/tasks/${id}/move`,
          {
            method: 'PATCH',
            body: JSON.stringify({ status, order_index: orderIndex }),
          },
        )
        const updated = res.data
        setTasks((prev) => prev.map((t) => (t.id === id ? updated : t)))
        return updated
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to move task')
        return null
      }
    },
    [],
  )

  const deleteTask = useCallback(async (id: string): Promise<boolean> => {
    try {
      await apiRequest<{ success: boolean }>(`/api/kanban/tasks/${id}`, {
        method: 'DELETE',
      })
      setTasks((prev) => prev.filter((t) => t.id !== id))
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete task')
      return false
    }
  }, [])

  const fetchComments = useCallback(async (taskId: string): Promise<KanbanComment[]> => {
    try {
      const res = await apiRequest<KanbanApiResponse<KanbanComment[]>>(
        `/api/kanban/tasks/${taskId}/comments`,
      )
      return res.data
    } catch {
      return []
    }
  }, [])

  const createComment = useCallback(
    async (taskId: string, content: string): Promise<KanbanComment | null> => {
      try {
        const res = await apiRequest<KanbanApiResponse<KanbanComment>>(
          `/api/kanban/tasks/${taskId}/comments`,
          {
            method: 'POST',
            body: JSON.stringify({ content }),
          },
        )
        return res.data
      } catch {
        return null
      }
    },
    [],
  )

  useEffect(() => {
    void fetchTasks()
  }, [fetchTasks])

  return {
    tasks,
    loading,
    error,
    fetchTasks,
    createTask,
    updateTask,
    moveTask,
    deleteTask,
    fetchComments,
    createComment,
  }
}
