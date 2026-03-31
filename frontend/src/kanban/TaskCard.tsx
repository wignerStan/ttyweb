import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { AlertTriangle, Calendar } from 'lucide-react'
import { type CSSProperties, memo } from 'react'
import { getTagColorClass } from './tagColors'
import type { KanbanTask } from './types'

interface TaskCardProps {
  task: KanbanTask
  onSelect: (task: KanbanTask) => void
}

const PRIORITY_CONFIG: Record<number, { label: string; className: string }> = {
  1: { label: 'Low', className: 'task-card__priority--low' },
  2: { label: 'Med', className: 'task-card__priority--medium' },
  3: { label: 'High', className: 'task-card__priority--high' },
  4: { label: 'Crit', className: 'task-card__priority--critical' },
  5: { label: 'Urg', className: 'task-card__priority--critical' },
} as const

const DEFAULT_PRIORITY = { label: '?', className: 'task-card__priority--medium' }

function getPriorityConfig(level: number) {
  return PRIORITY_CONFIG[level] ?? DEFAULT_PRIORITY
}

function formatDueDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

function isOverdue(dateStr: string): boolean {
  const due = new Date(dateStr)
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return due < today
}

function TaskCardInner({ task, onSelect }: TaskCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: task.id })

  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  }

  const priority = getPriorityConfig(task.priority)
  const dueDate = task.due_date

  return (
    <>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: dnd-kit requires div wrapper */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: dnd-kit requires div wrapper */}
      <div
        ref={setNodeRef}
        style={style}
        className={`task-card ${isDragging ? 'task-card--dragging' : ''}`}
        onClick={() => onSelect(task)}
        {...attributes}
      >
        <div className="task-card__header" ref={setActivatorNodeRef} {...listeners}>
          <span className="task-card__title">{task.title}</span>
          {priority && (
            <span
              className={`task-card__priority ${priority.className}`}
              title={`Priority: ${priority.label}`}
            >
              {priority.label}
            </span>
          )}
        </div>

        {task.tags.length > 0 && (
          <div className="task-card__tags">
            {task.tags.map((tag) => (
              <span key={tag} className={`task-card__tag ${getTagColorClass(tag)}`}>
                {tag}
              </span>
            ))}
          </div>
        )}

        {dueDate && (
          <div className="task-card__footer">
            <span
              className={`task-card__due ${isOverdue(dueDate) ? 'task-card__due--overdue' : ''}`}
            >
              {isOverdue(dueDate) ? <AlertTriangle size={10} /> : <Calendar size={10} />}
              {formatDueDate(dueDate)}
            </span>
          </div>
        )}
      </div>
    </>
  )
}

export type { TaskCardProps }
export const TaskCard = memo(TaskCardInner)
