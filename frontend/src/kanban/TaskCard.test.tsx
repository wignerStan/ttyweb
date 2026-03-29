import { screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../test-utils'
import { TaskCard } from './TaskCard'
import type { KanbanTask } from './types'

vi.mock('@dnd-kit/sortable', () => ({
  useSortable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    setActivatorNodeRef: vi.fn(),
    transform: null,
    transition: null,
    isDragging: false,
  }),
}))

vi.mock('@dnd-kit/utilities', () => ({
  CSS: {
    Transform: {
      toString: () => undefined,
    },
  },
}))

const baseTask: KanbanTask = {
  id: '1',
  title: 'Fix login bug',
  description: '',
  status: 'todo',
  priority: 3,
  tags: ['bug', 'feature'],
  due_date: '2099-06-15',
  order_index: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function renderCard(overrides: Partial<KanbanTask> = {}) {
  const task = { ...baseTask, ...overrides }
  const onSelect = vi.fn()
  const utils = renderWithProviders(<TaskCard task={task} onSelect={onSelect} />)
  return { task, onSelect, ...utils }
}

describe('TaskCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-29T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders task title', () => {
    renderCard()
    expect(screen.getByText('Fix login bug')).toBeInTheDocument()
  })

  it('shows priority badge with correct label', () => {
    renderCard({ priority: 1 })
    expect(screen.getByTitle('Priority: Low')).toBeInTheDocument()
  })

  it('shows tags with color classes', () => {
    renderCard()
    expect(screen.getByText('bug')).toHaveClass('tag--red')
    expect(screen.getByText('feature')).toHaveClass('tag--blue')
  })

  it('shows due date', () => {
    renderCard()
    expect(screen.getByText(/Jun 15/)).toBeInTheDocument()
  })

  it('shows overdue warning when past due', () => {
    renderCard({ due_date: '2026-01-01' })
    const dueEl = document.querySelector('.task-card__due--overdue')
    expect(dueEl).toBeInTheDocument()
  })

  it('clicking calls onSelect', () => {
    const { onSelect } = renderCard()
    screen.getByText('Fix login bug').click()
    expect(onSelect).toHaveBeenCalledOnce()
  })
})
