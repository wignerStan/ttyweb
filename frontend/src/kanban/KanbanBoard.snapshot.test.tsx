import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { KanbanBoard } from './KanbanBoard'
import type { KanbanTask } from './types'

const mockTasks: KanbanTask[] = [
  {
    id: '1',
    title: 'Set up CI',
    description: 'Add GitHub Actions',
    status: 'todo',
    priority: 1,
    tags: ['devops'],
    due_date: null,
    order_index: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: '2',
    title: 'Write tests',
    description: 'Cover core modules',
    status: 'in_progress',
    priority: 2,
    tags: ['testing'],
    due_date: '2026-02-01',
    order_index: 0,
    created_at: '2026-01-02T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
  },
]

vi.mock('./useKanbanTasks', () => ({
  useKanbanTasks: () => ({
    tasks: mockTasks,
    loading: false,
    error: null,
    fetchTasks: vi.fn(),
    createTask: vi.fn(),
    updateTask: vi.fn(),
    moveTask: vi.fn(),
    deleteTask: vi.fn(),
    fetchComments: vi.fn(),
    createComment: vi.fn(),
  }),
}))

describe('KanbanBoard snapshot', () => {
  it('renders board with tasks', () => {
    const { container } = render(<KanbanBoard />)
    expect(container).toMatchSnapshot()
  })
})
