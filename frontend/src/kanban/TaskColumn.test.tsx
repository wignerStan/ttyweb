import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { TaskColumn } from './TaskColumn'
import type { KanbanStatus, KanbanTask } from './types'

vi.mock('@dnd-kit/core', () => ({
  useDroppable: () => ({ setNodeRef: vi.fn(), isOver: false }),
}))

vi.mock('@dnd-kit/sortable', () => ({
  SortableContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  verticalListSortingStrategy: undefined,
}))

vi.mock('./TaskCard', () => ({
  TaskCard: ({ task, onSelect }: { task: KanbanTask; onSelect: (t: KanbanTask) => void }) => (
    <div data-testid="task-card" onClick={() => onSelect(task)}>
      {task.title}
    </div>
  ),
}))

const status: KanbanStatus = 'todo'

const tasks: KanbanTask[] = [
  {
    id: '1',
    title: 'Task A',
    description: '',
    status: 'todo',
    priority: 2,
    tags: [],
    due_date: null,
    order_index: 0,
    created_at: '',
    updated_at: '',
  },
  {
    id: '2',
    title: 'Task B',
    description: '',
    status: 'todo',
    priority: 3,
    tags: [],
    due_date: null,
    order_index: 1,
    created_at: '',
    updated_at: '',
  },
]

describe('TaskColumn', () => {
  it('renders column title and badge count', () => {
    renderWithProviders(
      <TaskColumn
        status={status}
        title="To Do"
        tasks={tasks}
        onSelectTask={vi.fn()}
        onAddTask={vi.fn()}
      />,
    )
    expect(screen.getByText('To Do')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('"Add task" button calls onAddTask', async () => {
    const onAddTask = vi.fn()
    renderWithProviders(
      <TaskColumn
        status={status}
        title="To Do"
        tasks={tasks}
        onSelectTask={vi.fn()}
        onAddTask={onAddTask}
      />,
    )
    const user = userEvent.setup()
    await user.click(screen.getByTitle('Add task to To Do'))
    expect(onAddTask).toHaveBeenCalledWith('todo')
  })

  it('renders child TaskCards', () => {
    renderWithProviders(
      <TaskColumn
        status={status}
        title="To Do"
        tasks={tasks}
        onSelectTask={vi.fn()}
        onAddTask={vi.fn()}
      />,
    )
    expect(screen.getAllByTestId('task-card')).toHaveLength(2)
  })
})
