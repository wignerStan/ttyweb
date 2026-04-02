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
  1: { label: 'Low', className: 'badge badge-xs badge-success' },
  2: { label: 'Med', className: 'badge badge-xs badge-warning' },
  3: { label: 'High', className: 'badge badge-xs badge-error' },
  4: { label: 'Crit', className: 'badge badge-xs badge-error' },
  5: { label: 'Urg', className: 'badge badge-xs badge-error' },
} as const

const DEFAULT_PRIORITY = { label: '?', className: 'badge badge-xs badge-warning' }

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
      {/* biome-ignore lint/a11y/useSemanticElements: dnd-kit requires div wrapper */}
      <div
        ref={setNodeRef}
        style={style}
        className={`task-card bg-base-100 rounded-box border border-base-200 p-3 shadow-sm cursor-grab select-none transition-all duration-150 hover:shadow-md active:cursor-grabbing ${
          isDragging ? 'task-card--dragging rotate-2 opacity-70 shadow-lg' : ''
        }`}
        onClick={() => onSelect(task)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onSelect(task)
          }
        }}
        {...attributes}
        role="button"
        tabIndex={0}
      >
        <div
          className="mb-1.5 flex items-start justify-between gap-2"
          ref={setActivatorNodeRef}
          {...listeners}
        >
          <span className="line-clamp-2 break-words text-xs font-medium leading-snug text-base-content overflow-hidden">
            {task.title}
          </span>
          {priority && (
            <span
              className={`flex-shrink-0 ${priority.className}`}
              title={`Priority: ${priority.label}`}
            >
              {priority.label}
            </span>
          )}
        </div>

        {task.tags.length > 0 && (
          <div className="mb-1.5 flex flex-wrap gap-1">
            {task.tags.map((tag) => (
              <span
                key={tag}
                className={`badge badge-sm whitespace-nowrap ${getTagColorClass(tag)}`}
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        {dueDate && (
          <div className="flex items-center gap-2">
            <span
              data-testid="task-card-due"
              className={`flex items-center gap-0.5 text-xs text-base-content/50 ${
                isOverdue(dueDate) ? 'text-error' : ''
              }`}
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
