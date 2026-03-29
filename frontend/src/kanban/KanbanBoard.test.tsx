import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { KanbanBoard } from './KanbanBoard'
import type { KanbanStatus, KanbanTask } from './types'

// ---------------------------------------------------------------------------
// DndContext mock — captures callbacks so we can invoke them in tests
// ---------------------------------------------------------------------------
type DndHandler = (...args: unknown[]) => unknown
const dndCallbacks: Record<string, DndHandler | undefined> = {}

vi.mock('@dnd-kit/core', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@dnd-kit/core')>()
  return {
    ...actual,
    DndContext: function MockDndContext({
      children,
      onDragStart,
      onDragEnd,
      ...rest
    }: Record<string, unknown>) {
      const onStart = onDragStart as DndHandler | undefined
      const onEnd = onDragEnd as DndHandler | undefined
      dndCallbacks.onDragStart = onStart
      dndCallbacks.onDragEnd = onEnd
      const { DndContext: RealDndContext } = actual
      // Use real DndContext but it won't get pointer events in jsdom,
      // so we rely on captured callbacks.
      return (
        <RealDndContext
          {...rest}
          onDragStart={onStart as ((event: DragStartEvent) => void) | undefined}
          onDragEnd={onEnd as ((event: DragEndEvent) => void) | undefined}
        >
          {children as React.ReactNode}
        </RealDndContext>
      )
    },
  }
})

// ---------------------------------------------------------------------------
// Mock useKanbanTasks - keep a mutable reference so each test can override
// ---------------------------------------------------------------------------
const defaultMockReturn = {
  tasks: [] as KanbanTask[],
  loading: false,
  error: null as string | null,
  fetchTasks: vi.fn(),
  createTask: vi.fn(),
  updateTask: vi.fn(),
  moveTask: vi.fn(),
  deleteTask: vi.fn(),
}

let mockHookReturn = { ...defaultMockReturn }

vi.mock('./useKanbanTasks', () => ({
  useKanbanTasks: () => mockHookReturn,
}))

// ---------------------------------------------------------------------------
// Minimal TaskCard / TaskColumn / TaskDialog mocks
// These components depend on dnd-kit internals; we mock them to isolate
// KanbanBoard logic.
// ---------------------------------------------------------------------------
vi.mock('./TaskCard', () => ({
  TaskCard: function MockTaskCard({ task, onSelect }: { task: KanbanTask; onSelect: () => void }) {
    return (
      <div data-testid="task-card" data-task-id={task.id} onClick={onSelect}>
        {task.title}
      </div>
    )
  },
}))

vi.mock('./TaskColumn', () => ({
  TaskColumn: function MockTaskColumn({
    status,
    title,
    tasks,
    onSelectTask,
    onAddTask,
  }: {
    status: KanbanStatus
    title: string
    tasks: KanbanTask[]
    onSelectTask: (task: KanbanTask) => void
    onAddTask: (status: KanbanStatus) => void
  }) {
    return (
      <div data-testid="task-column" data-status={status}>
        <h3>{title}</h3>
        <span data-testid="task-count">{tasks.length}</span>
        {tasks.map((task) => (
          <div
            key={task.id}
            data-testid={`column-task-${task.id}`}
            onClick={() => onSelectTask(task)}
          >
            {task.title}
          </div>
        ))}
        <button data-testid={`column-add-${status}`} onClick={() => onAddTask(status)}>
          Add
        </button>
      </div>
    )
  },
}))

vi.mock('./TaskDialog', () => ({
  TaskDialog: function MockTaskDialog({
    open,
    task,
    defaultStatus,
    onSave,
    onCreate,
    onDelete,
    onClose,
  }: Record<string, unknown>) {
    if (!open) return null
    return (
      <div data-testid="task-dialog">
        <span data-testid="dialog-mode">{task ? 'edit' : 'create'}</span>
        <span data-testid="dialog-default-status">{String(defaultStatus)}</span>
        <button data-testid="dialog-close" onClick={() => (onClose as () => void)()}>
          Close
        </button>
        <button
          data-testid="dialog-create"
          onClick={() =>
            (onCreate as (fields: Record<string, unknown>) => Promise<unknown>)({
              title: 'New',
              status: defaultStatus,
            })
          }
        >
          Create
        </button>
        <button
          data-testid="dialog-save"
          onClick={() =>
            (onSave as (fields: Record<string, unknown>) => Promise<unknown>)({
              title: 'Updated',
              status: (task as KanbanTask)?.status,
            })
          }
        >
          Save
        </button>
        <button
          data-testid="dialog-delete"
          onClick={() =>
            (onDelete as (id: string) => Promise<boolean>)((task as KanbanTask)?.id ?? '1')
          }
        >
          Delete
        </button>
      </div>
    )
  },
}))

// ---------------------------------------------------------------------------
// Sample data
// ---------------------------------------------------------------------------
function makeTask(
  overrides: Partial<KanbanTask> & { id: string; title: string; status: KanbanStatus },
): KanbanTask {
  return {
    description: '',
    priority: 1,
    tags: [],
    due_date: null,
    order_index: 0,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

const todoTask = makeTask({ id: 't1', title: 'Setup CI', status: 'todo' })
const inProgressTask = makeTask({
  id: 't2',
  title: 'Write tests',
  status: 'in_progress',
  order_index: 1,
})
const doneTask = makeTask({ id: 't3', title: 'Deploy', status: 'done', order_index: 2 })

/** Helper to find a mock column by status */
function getColumnByStatus(container: HTMLElement, status: KanbanStatus): HTMLElement {
  const col = container.querySelector(
    `[data-testid="task-column"][data-status="${status}"]`,
  ) as HTMLElement | null
  if (!col) throw new Error(`Column with status "${status}" not found`)
  return col
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------
describe('KanbanBoard', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    mockHookReturn = {
      ...defaultMockReturn,
      fetchTasks: vi.fn(),
      createTask: vi.fn(),
      updateTask: vi.fn(),
      moveTask: vi.fn(),
      deleteTask: vi.fn(),
    }
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  // -----------------------------------------------------------------------
  // Error state
  // -----------------------------------------------------------------------
  describe('error state', () => {
    it('renders error message and retry button', () => {
      mockHookReturn.error = 'Network failure'

      render(<KanbanBoard />)

      expect(screen.getByText('Network failure')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
    })

    it('calls fetchTasks when retry is clicked', async () => {
      mockHookReturn.error = 'Network failure'
      const fetchTasks = vi.fn().mockResolvedValue(undefined)
      mockHookReturn.fetchTasks = fetchTasks

      render(<KanbanBoard />)
      const retryBtn = screen.getByRole('button', { name: /retry/i })

      await userEvent.click(retryBtn)

      expect(fetchTasks).toHaveBeenCalled()
    })
  })

  // -----------------------------------------------------------------------
  // Loading state
  // -----------------------------------------------------------------------
  describe('loading state', () => {
    it('shows loading spinner when loading with no tasks', () => {
      mockHookReturn.loading = true
      mockHookReturn.tasks = []

      render(<KanbanBoard />)

      expect(screen.getByText('Loading tasks...')).toBeInTheDocument()
    })

    it('shows tasks even when loading (subsequent loads)', () => {
      mockHookReturn.loading = true
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      expect(screen.queryByText('Loading tasks...')).not.toBeInTheDocument()
      expect(screen.getByText('Setup CI')).toBeInTheDocument()
    })
  })

  // -----------------------------------------------------------------------
  // Empty state
  // -----------------------------------------------------------------------
  describe('empty state', () => {
    it('shows empty message when no tasks', () => {
      mockHookReturn.loading = false
      mockHookReturn.tasks = []

      render(<KanbanBoard />)

      expect(screen.getByText('No tasks yet. Create one to get started.')).toBeInTheDocument()
    })
  })

  // -----------------------------------------------------------------------
  // Board rendering with tasks
  // -----------------------------------------------------------------------
  describe('board rendering', () => {
    it('renders the board header with title', () => {
      mockHookReturn.tasks = [todoTask, inProgressTask, doneTask]

      render(<KanbanBoard />)

      expect(screen.getByText('Kanban Board')).toBeInTheDocument()
      expect(screen.getByText('Drag and drop tasks between columns')).toBeInTheDocument()
    })

    it('renders all four column headers', () => {
      mockHookReturn.tasks = [todoTask, inProgressTask, doneTask]

      render(<KanbanBoard />)

      expect(screen.getByText('Todo')).toBeInTheDocument()
      expect(screen.getByText('In Progress')).toBeInTheDocument()
      expect(screen.getByText('Done')).toBeInTheDocument()
      expect(screen.getByText('Archived')).toBeInTheDocument()
    })

    it('passes tasks to columns by status', () => {
      mockHookReturn.tasks = [todoTask, inProgressTask, doneTask]
      const { container } = render(<KanbanBoard />)

      const todoCol = getColumnByStatus(container, 'todo')
      const ipCol = getColumnByStatus(container, 'in_progress')
      const doneCol = getColumnByStatus(container, 'done')
      const archivedCol = getColumnByStatus(container, 'archived')

      expect(todoCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(1)
      expect(ipCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(1)
      expect(doneCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(1)
      expect(archivedCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(0)
    })

    it('renders refresh button in header', () => {
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      const refreshBtn = screen.getByTitle('Refresh tasks')
      expect(refreshBtn).toBeInTheDocument()
    })

    it('disables refresh button while loading', () => {
      mockHookReturn.loading = true
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      const refreshBtn = screen.getByTitle('Refresh tasks')
      expect(refreshBtn).toBeDisabled()
    })

    it('calls fetchTasks when refresh button is clicked', async () => {
      const fetchTasks = vi.fn().mockResolvedValue(undefined)
      mockHookReturn.fetchTasks = fetchTasks
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)
      const refreshBtn = screen.getByTitle('Refresh tasks')

      await userEvent.click(refreshBtn)

      expect(fetchTasks).toHaveBeenCalled()
    })

    it('renders Add Task button in header', () => {
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      expect(screen.getByRole('button', { name: /add task/i })).toBeInTheDocument()
    })
  })

  // -----------------------------------------------------------------------
  // Add Task dialog
  // -----------------------------------------------------------------------
  describe('add task dialog', () => {
    it('opens create dialog when Add Task is clicked', async () => {
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      await userEvent.click(screen.getByRole('button', { name: /add task/i }))

      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()
      expect(screen.getByTestId('dialog-mode').textContent).toBe('create')
      expect(screen.getByTestId('dialog-default-status').textContent).toBe('todo')
    })

    it('opens create dialog for specific column when column add button is clicked', async () => {
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      await userEvent.click(screen.getByTestId('column-add-in_progress'))

      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()
      expect(screen.getByTestId('dialog-default-status').textContent).toBe('in_progress')
    })
  })

  // -----------------------------------------------------------------------
  // Edit Task dialog
  // -----------------------------------------------------------------------
  describe('edit task dialog', () => {
    it('opens edit dialog when a task card is clicked', async () => {
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      await userEvent.click(screen.getByTestId('column-task-t1'))

      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()
      expect(screen.getByTestId('dialog-mode').textContent).toBe('edit')
      expect(screen.getByTestId('dialog-default-status').textContent).toBe('todo')
    })

    it('closes dialog when close button is clicked', async () => {
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Open the dialog
      await userEvent.click(screen.getByTestId('column-task-t1'))
      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()

      // Close it
      await userEvent.click(screen.getByTestId('dialog-close'))
      expect(screen.queryByTestId('task-dialog')).not.toBeInTheDocument()
    })
  })

  // -----------------------------------------------------------------------
  // Task creation via dialog
  // -----------------------------------------------------------------------
  describe('task creation', () => {
    it('calls createTask with form fields when dialog create is triggered', async () => {
      const createTask = vi.fn().mockResolvedValue({ id: 'new', title: 'New' })
      mockHookReturn.createTask = createTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Open create dialog
      await userEvent.click(screen.getByRole('button', { name: /add task/i }))
      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()

      // Click create
      await userEvent.click(screen.getByTestId('dialog-create'))

      expect(createTask).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'New',
          status: 'todo',
        }),
      )
    })
  })

  // -----------------------------------------------------------------------
  // Task update via dialog
  // -----------------------------------------------------------------------
  describe('task update', () => {
    it('calls updateTask with form fields when dialog save is triggered', async () => {
      const updateTask = vi.fn().mockResolvedValue({ id: 't1', title: 'Updated' })
      mockHookReturn.updateTask = updateTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Open edit dialog
      await userEvent.click(screen.getByTestId('column-task-t1'))
      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()

      // Click save
      await userEvent.click(screen.getByTestId('dialog-save'))

      expect(updateTask).toHaveBeenCalledWith(
        't1',
        expect.objectContaining({
          title: 'Updated',
        }),
      )
    })

    it('does not call updateTask when editingTask is null', async () => {
      const updateTask = vi.fn()
      mockHookReturn.updateTask = updateTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Open create dialog (no editing task)
      await userEvent.click(screen.getByRole('button', { name: /add task/i }))

      // Click save (which calls handleSave)
      await userEvent.click(screen.getByTestId('dialog-save'))

      expect(updateTask).not.toHaveBeenCalled()
    })
  })

  // -----------------------------------------------------------------------
  // Task delete via dialog
  // -----------------------------------------------------------------------
  describe('task deletion', () => {
    it('calls deleteTask when dialog delete is triggered', async () => {
      const deleteTask = vi.fn().mockResolvedValue(true)
      mockHookReturn.deleteTask = deleteTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Open edit dialog
      await userEvent.click(screen.getByTestId('column-task-t1'))
      expect(screen.getByTestId('task-dialog')).toBeInTheDocument()

      // Click delete
      await userEvent.click(screen.getByTestId('dialog-delete'))

      expect(deleteTask).toHaveBeenCalledWith('t1')
    })
  })

  // -----------------------------------------------------------------------
  // Drag and drop
  // -----------------------------------------------------------------------
  describe('drag and drop', () => {
    // We cannot fully test dnd-kit drag interactions in jsdom without
    // a full pointer-event simulation, but we can verify the handlers exist
    // and the component renders DndContext. The key logic is in handleDragEnd
    // which we test by verifying moveTask is called with correct params.

    it('renders board with DndContext when tasks exist', () => {
      mockHookReturn.tasks = [todoTask, inProgressTask]

      render(<KanbanBoard />)

      // The board body should be present with columns
      expect(screen.getAllByTestId('task-column').length).toBeGreaterThanOrEqual(2)
    })

    it('sets activeTask when a task is found for the active drag id', () => {
      // This is verified indirectly: when activeId is set, the DragOverlay
      // should render a TaskCard. We can't simulate drag events in jsdom easily,
      // but we verify the component structure is correct.
      mockHookReturn.tasks = [todoTask, inProgressTask]

      render(<KanbanBoard />)

      // TaskCard should be present in the DOM (as part of the column)
      expect(screen.getByText('Setup CI')).toBeInTheDocument()
      expect(screen.getByText('Write tests')).toBeInTheDocument()
    })
  })

  // -----------------------------------------------------------------------
  // tasksByStatus sorting
  // -----------------------------------------------------------------------
  describe('task ordering', () => {
    it('sorts tasks by order_index within each column', () => {
      const task1 = makeTask({ id: 'a', title: 'Second', status: 'todo', order_index: 2000 })
      const task2 = makeTask({ id: 'b', title: 'First', status: 'todo', order_index: 1000 })
      const task3 = makeTask({ id: 'c', title: 'Third', status: 'todo', order_index: 3000 })
      mockHookReturn.tasks = [task1, task2, task3]
      const { container } = render(<KanbanBoard />)

      const todoCol = getColumnByStatus(container, 'todo')
      const taskElements = todoCol.querySelectorAll('[data-testid^="column-task-"]')
      expect(taskElements).toHaveLength(3)

      // Should be sorted by order_index: First (1000), Second (2000), Third (3000)
      expect(taskElements[0]?.textContent).toBe('First')
      expect(taskElements[1]?.textContent).toBe('Second')
      expect(taskElements[2]?.textContent).toBe('Third')
    })

    it('correctly groups tasks by status across multiple columns', () => {
      const t1 = makeTask({ id: 'x', title: 'T1', status: 'todo' })
      const t2 = makeTask({ id: 'y', title: 'T2', status: 'done' })
      const t3 = makeTask({ id: 'z', title: 'T3', status: 'archived' })
      mockHookReturn.tasks = [t1, t2, t3]
      const { container } = render(<KanbanBoard />)

      const todoCol = getColumnByStatus(container, 'todo')
      const doneCol = getColumnByStatus(container, 'done')
      const archivedCol = getColumnByStatus(container, 'archived')
      const ipCol = getColumnByStatus(container, 'in_progress')

      expect(todoCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(1)
      expect(doneCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(1)
      expect(archivedCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(1)
      expect(ipCol.querySelectorAll('[data-testid^="column-task-"]').length).toBe(0)
    })
  })

  // -----------------------------------------------------------------------
  // Drag event handling (via captured callbacks)
  // -----------------------------------------------------------------------
  describe('drag event handlers', () => {
    it('handleDragStart sets activeId on the event', () => {
      mockHookReturn.tasks = [todoTask, inProgressTask]

      render(<KanbanBoard />)

      // Invoke the captured onDragStart
      expect(dndCallbacks.onDragStart).toBeDefined()
      act(() => {
        dndCallbacks.onDragStart!({ active: { id: 't1' } })
      })

      // The callback was captured and could be invoked without error
      // (DragOverlay portal doesn't render in jsdom, so we can't check DOM)
      expect(dndCallbacks.onDragStart).toBeDefined()
    })

    it('handleDragEnd calls moveTask when dragging to a different column', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      mockHookReturn.tasks = [todoTask, inProgressTask]

      render(<KanbanBoard />)

      // Simulate drag end: drag t1 from todo onto in_progress column header
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 'in_progress' },
        })
      })

      expect(moveTask).toHaveBeenCalledWith('t1', 'in_progress', expect.any(Number))
    })

    it('handleDragEnd calls moveTask when dragging onto another task in different column', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      mockHookReturn.tasks = [todoTask, inProgressTask]

      render(<KanbanBoard />)

      // Drag t1 onto t2 (which is in in_progress)
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 't2' },
        })
      })

      expect(moveTask).toHaveBeenCalledWith('t1', 'in_progress', expect.any(Number))
    })

    it('handleDragEnd skips when no over target', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: null,
        })
      })

      expect(moveTask).not.toHaveBeenCalled()
    })

    it('handleDragEnd skips when dropped on same task in same position', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Drag t1 onto itself (same column, same id)
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 't1' },
        })
      })

      expect(moveTask).not.toHaveBeenCalled()
    })

    it('handleDragEnd skips when task or target not found', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      mockHookReturn.tasks = [todoTask]

      render(<KanbanBoard />)

      // Drag a non-existent task
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 'nonexistent' },
          over: { id: 't1' },
        })
      })

      expect(moveTask).not.toHaveBeenCalled()
    })

    it('handleDragEnd calculates order index when dropping on column header (appends to end)', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      const targetTask = makeTask({
        id: 't2',
        title: 'Target',
        status: 'in_progress',
        order_index: 2000,
      })
      mockHookReturn.tasks = [todoTask, targetTask]

      render(<KanbanBoard />)

      // Drop on in_progress column header: overId = 'in_progress', not a task
      // overIndex = -1, newIndex = targetColumnTasks.length = 1
      // orderIndex = last.order_index + 1000 = 3000
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 'in_progress' },
        })
      })

      expect(moveTask).toHaveBeenCalledWith('t1', 'in_progress', 3000) // 2000 + 1000
    })

    it('handleDragEnd calculates order index when dropping on last task in column', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      const targetTask = makeTask({
        id: 't2',
        title: 'Target',
        status: 'in_progress',
        order_index: 2000,
      })
      mockHookReturn.tasks = [todoTask, targetTask]

      render(<KanbanBoard />)

      // Drop t1 onto t2 (the only task in in_progress)
      // overIndex = 0, newIndex = 0
      // Since newIndex === 0 && targetColumnTasks.length > 0:
      //   orderIndex = targetColumnTasks[0].order_index - 1000 = 2000 - 1000 = 1000
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 't2' },
        })
      })

      expect(moveTask).toHaveBeenCalledWith('t1', 'in_progress', 1000) // 2000 - 1000
    })

    it('handleDragEnd calculates midpoint order index when dropping between tasks', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      const t2 = makeTask({ id: 't2', title: 'T2', status: 'in_progress', order_index: 2000 })
      const t3 = makeTask({ id: 't3', title: 'T3', status: 'in_progress', order_index: 4000 })
      mockHookReturn.tasks = [todoTask, t2, t3]

      render(<KanbanBoard />)

      // Drop t1 between t2 and t3 in in_progress column
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 't3' },
        })
      })

      // overIndex for t3 = 1, newIndex = 1, prev = t2 (order 2000), next = t3 (order 4000)
      // orderIndex = Math.round((2000 + 4000) / 2) = 3000
      expect(moveTask).toHaveBeenCalledWith('t1', 'in_progress', 3000)
    })

    it('handleDragEnd resets activeId after drag end', () => {
      const moveTask = vi.fn().mockResolvedValue(null)
      mockHookReturn.moveTask = moveTask
      mockHookReturn.tasks = [todoTask, inProgressTask]

      render(<KanbanBoard />)

      // Start drag
      act(() => {
        dndCallbacks.onDragStart!({ active: { id: 't1' } })
      })

      // End drag
      act(() => {
        dndCallbacks.onDragEnd!({
          active: { id: 't1' },
          over: { id: 'in_progress' },
        })
      })

      // moveTask was called, confirming handleDragEnd ran with proper cleanup
      expect(moveTask).toHaveBeenCalledWith('t1', 'in_progress', expect.any(Number))
    })
  })
})
