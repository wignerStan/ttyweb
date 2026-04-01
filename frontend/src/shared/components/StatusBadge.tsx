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
  idle: 'text-on-surface-muted',
  in_progress: 'text-primary',
  done: 'text-success',
  failed: 'text-error',
  waiting: 'text-warning',
}

const sizeClasses = {
  small: 'text-[0.7rem]',
  medium: 'text-[0.85rem]',
} as const

function StatusIcon({ status, size }: { status: PaneStatus; size: 'small' | 'medium' }) {
  const iconSize = size === 'small' ? 10 : 12

  if (status === 'in_progress') {
    return <Loader2 size={iconSize} className="shrink-0 text-primary animate-spin-slow" />
  }
  if (status === 'done') {
    return <Check size={iconSize} className="shrink-0 text-success" />
  }
  if (status === 'failed') {
    return <XCircle size={iconSize} className="shrink-0 text-error" />
  }
  if (status === 'waiting') {
    return <Clock size={iconSize} className="shrink-0 text-warning" />
  }
  return <Circle size={iconSize} className="shrink-0 text-on-surface-muted" />
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
          className="appearance-none bg-transparent border-none text-inherit text-xs cursor-pointer p-px pr-3 rounded [&>option]:bg-base-200 [&>option]:text-base-content"
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
