import { Check, Circle, Clock, Loader2, XCircle } from 'lucide-react'
import type { PaneStatus } from '../../types'

interface Props {
  status: PaneStatus
  onChange?: (newStatus: PaneStatus) => void
  size?: 'small' | 'medium'
}

const statusLabels: Record<PaneStatus, string> = {
  idle: 'Idle',
  in_progress: 'In Progress',
  done: 'Done',
  failed: 'Failed',
  waiting: 'Waiting',
}

const statusOptions: PaneStatus[] = ['idle', 'in_progress', 'done', 'failed', 'waiting']

const statusColorClasses: Record<PaneStatus, string> = {
  idle: 'text-[var(--zinc-500)]',
  in_progress: 'text-[var(--blue-500)]',
  done: 'text-[var(--green-500)]',
  failed: 'text-[var(--red-500)]',
  waiting: 'text-[var(--yellow-500)]',
}

const sizeClasses = {
  small: 'text-[0.7rem]',
  medium: 'text-[0.85rem]',
} as const

function StatusIcon({ status, size }: { status: PaneStatus; size: 'small' | 'medium' }) {
  const iconSize = size === 'small' ? 10 : 12

  if (status === 'in_progress') {
    return <Loader2 size={iconSize} className="shrink-0 text-[var(--blue-500)] animate-spin-slow" />
  }
  if (status === 'done') {
    return <Check size={iconSize} className="shrink-0 text-[var(--green-500)]" />
  }
  if (status === 'failed') {
    return <XCircle size={iconSize} className="shrink-0 text-[var(--red-500)]" />
  }
  if (status === 'waiting') {
    return <Clock size={iconSize} className="shrink-0 text-[var(--yellow-500)]" />
  }
  return <Circle size={iconSize} className="shrink-0 text-[var(--zinc-600)]" />
}

export function StatusBadge({ status, onChange, size = 'small' }: Props) {
  const isEditable = !!onChange

  if (isEditable) {
    return (
      <div
        className={`inline-flex items-center gap-1 font-inherit ${statusColorClasses[status]} ${sizeClasses[size]} relative cursor-pointer p-0.5 px-1 rounded transition-colors duration-150 hover:bg-white/[0.08]`}
      >
        <StatusIcon status={status} size={size} />
        <select
          className="appearance-none bg-transparent border-none text-inherit text-xs cursor-pointer p-px pr-3 rounded [&>option]:bg-[var(--zinc-900)] [&>option]:text-[var(--zinc-200)]"
          value={status}
          onChange={(e) => onChange(e.target.value as PaneStatus)}
          onClick={(e) => e.stopPropagation()}
        >
          {statusOptions.map((opt) => (
            <option key={opt} value={opt}>
              {statusLabels[opt]}
            </option>
          ))}
        </select>
      </div>
    )
  }

  return (
    <div
      className={`inline-flex items-center gap-1 font-inherit ${statusColorClasses[status]} ${sizeClasses[size]}`}
    >
      <StatusIcon status={status} size={size} />
      {size === 'medium' && (
        <span className="font-medium tracking-[0.02em]">{statusLabels[status]}</span>
      )}
    </div>
  )
}
