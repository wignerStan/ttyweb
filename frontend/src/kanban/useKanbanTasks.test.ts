import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, type Mock, vi } from 'vitest'
import type { KanbanApiResponse, KanbanComment, KanbanTask } from './types'
import { useKanbanTasks } from './useKanbanTasks'

// --- Helpers ---

function makeTask(overrides: Partial<KanbanTask> = {}): KanbanTask {
  return {
    id: 'task-1',
    title: 'Test Task',
    description: '',
    status: 'todo',
    priority: 0,
    tags: [],
    due_date: null,
    order_index: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function makeComment(overrides: Partial<KanbanComment> = {}): KanbanComment {
  return {
    id: 'cmt-1',
    task_id: 'task-1',
    content: 'Nice task',
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function mockFetchOk<T>(data: T): Mock {
  return vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve({ success: true, data } satisfies KanbanApiResponse<T>),
  })
}

function mockFetchError(status: number, body = ''): Mock {
  return vi.fn().mockResolvedValue({
    ok: false,
    status,
    text: () => Promise.resolve(body),
    json: () => Promise.resolve({ success: false, error: body }),
  })
}

// --- Tests ---

describe('useKanbanTasks', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch tasks across all statuses on mount', async () => {
    const todoTask = makeTask({ id: '1', status: 'todo' })
    const doneTask = makeTask({ id: '2', status: 'done' })

    const statusResponses: Record<string, KanbanTask[]> = {
      todo: [todoTask],
      in_progress: [],
      done: [doneTask],
      archived: [],
    }

    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      const status = url.split('status=')[1]!
      const tasks = statusResponses[status] ?? []
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ success: true, data: tasks }),
      })
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.tasks).toHaveLength(2)
    expect(result.current.tasks).toEqual(expect.arrayContaining([todoTask, doneTask]))
    expect(globalThis.fetch).toHaveBeenCalledTimes(4)
  })

  it('should handle fetch failure for individual statuses gracefully', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      const status = url.split('status=')[1]!
      if (status === 'todo') {
        return Promise.resolve({
          ok: false,
          status: 500,
          text: () => Promise.resolve('server error'),
        })
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ success: true, data: [] }),
      })
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.tasks).toEqual([])
    expect(result.current.error).toBeNull()
  })

  it('should create a task and add it to state', async () => {
    const newTask = makeTask({ id: 'new', title: 'New Task' })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      // Initial fetch: return empty for all statuses
      if (url.includes('/tasks?status=')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchOk(newTask)()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchOk(newTask)

    let created: KanbanTask | null = null
    await act(async () => {
      created = await result.current.createTask({ title: 'New Task' })
    })

    expect(created).toEqual(newTask)
    expect(result.current.tasks).toContainEqual(newTask)
  })

  it('should return null and set error on create failure', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchError(500, 'creation failed')()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchError(500, 'creation failed')

    let created: KanbanTask | null = null
    await act(async () => {
      created = await result.current.createTask({ title: 'Bad' })
    })

    expect(created).toBeNull()
    expect(result.current.error).toContain('API 500')
  })

  it('should update a task and replace it in state', async () => {
    const original = makeTask({ id: '1', title: 'Original' })
    const updated = makeTask({ id: '1', title: 'Updated' })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        // Only return the task for 'todo' status, empty for others
        if (url.includes('status=todo')) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ success: true, data: [original] }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchOk(updated)()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(1)
    })

    globalThis.fetch = mockFetchOk(updated)

    await act(async () => {
      await result.current.updateTask('1', { title: 'Updated' })
    })

    expect(result.current.tasks).toHaveLength(1)
    expect(result.current.tasks[0]!.title).toBe('Updated')
  })

  it('should return null on update failure', async () => {
    const task = makeTask({ id: '1' })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        if (url.includes('status=todo')) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ success: true, data: [task] }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchError(404, 'not found')()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(1)
    })

    globalThis.fetch = mockFetchError(404, 'not found')

    let updated: KanbanTask | null = null
    await act(async () => {
      updated = await result.current.updateTask('1', { title: 'X' })
    })

    expect(updated).toBeNull()
    expect(result.current.tasks[0]!.title).toBe('Test Task')
  })

  it('should move a task to a new status', async () => {
    const task = makeTask({ id: '1', status: 'todo', order_index: 0 })
    const moved = makeTask({ id: '1', status: 'done', order_index: 2 })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        if (url.includes('status=todo')) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ success: true, data: [task] }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchOk(moved)()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(1)
    })

    globalThis.fetch = mockFetchOk(moved)

    await act(async () => {
      await result.current.moveTask('1', 'done', 2)
    })

    expect(result.current.tasks[0]!.status).toBe('done')
    expect(result.current.tasks[0]!.order_index).toBe(2)
  })

  it('should delete a task and remove it from state', async () => {
    const task = makeTask({ id: '1' })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        if (url.includes('status=todo')) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ success: true, data: [task] }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchOk({ success: true })()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(1)
    })

    globalThis.fetch = mockFetchOk({ success: true })

    let deleted!: boolean
    await act(async () => {
      deleted = await result.current.deleteTask('1')
    })

    expect(deleted).toBe(true)
    expect(result.current.tasks).toHaveLength(0)
  })

  it('should return false on delete failure', async () => {
    const task = makeTask({ id: '1' })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        if (url.includes('status=todo')) {
          return Promise.resolve({
            ok: true,
            json: () => Promise.resolve({ success: true, data: [task] }),
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchError(403, 'forbidden')()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(1)
    })

    globalThis.fetch = mockFetchError(403, 'forbidden')

    let deleted!: boolean
    await act(async () => {
      deleted = await result.current.deleteTask('1')
    })

    expect(deleted).toBe(false)
    expect(result.current.tasks).toHaveLength(1)
  })

  it('should fetch comments for a task', async () => {
    const comments = [makeComment({ id: 'c1' }), makeComment({ id: 'c2' })]
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchOk(comments)()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchOk(comments)

    let fetched!: KanbanComment[]
    await act(async () => {
      fetched = await result.current.fetchComments('task-1')
    })

    expect(fetched).toEqual(comments)
  })

  it('should return empty array when fetchComments fails', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchError(500, 'fail')()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchError(500, 'fail')

    let comments!: KanbanComment[]
    await act(async () => {
      comments = await result.current.fetchComments('task-1')
    })

    expect(comments).toEqual([])
  })

  it('should create a comment and return it', async () => {
    const comment = makeComment({ content: 'Hello' })
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchOk(comment)()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchOk(comment)

    let created!: KanbanComment | null
    await act(async () => {
      created = await result.current.createComment('task-1', 'Hello')
    })

    expect(created).toEqual(comment)
  })

  it('should return null when createComment fails', async () => {
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/tasks?status=')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ success: true, data: [] }),
        })
      }
      return mockFetchError(400, 'bad request')()
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    globalThis.fetch = mockFetchError(400, 'bad request')

    let created!: KanbanComment | null
    await act(async () => {
      created = await result.current.createComment('task-1', '')
    })

    expect(created).toBeNull()
  })

  it('should manually refetch tasks via fetchTasks', async () => {
    const first = [makeTask({ id: '1' })]
    const second = [makeTask({ id: '1' }), makeTask({ id: '2' })]
    let callCount = 0

    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++
      const data = callCount <= 4 ? first : second
      const statusIdx = (callCount - 1) % 4
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ success: true, data: statusIdx === 0 ? data : [] }),
      })
    })

    const { result } = renderHook(() => useKanbanTasks())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.tasks).toHaveLength(1)

    await act(async () => {
      await result.current.fetchTasks()
    })

    await waitFor(() => {
      expect(result.current.tasks).toHaveLength(2)
    })
  })
})
