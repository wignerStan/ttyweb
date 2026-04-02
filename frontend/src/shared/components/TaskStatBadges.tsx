import { CheckCircle2, Clock, Loader2, RefreshCw, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { getAuthHeaders } from '../../utils/auth'

interface TaskStatBadgesProps {
  refreshToken?: number
}

export function TaskStatBadges({ refreshToken: _refreshToken }: TaskStatBadgesProps) {
  const [inProgress, setInProgress] = useState(0)
  const [completed, setCompleted] = useState(0)
  const [failed, setFailed] = useState(0)
  const [waiting, setWaiting] = useState(0)
  const [loading, setLoading] = useState(false)

  const fetch_ = useCallback(async () => {
    setLoading(true)
    try {
      const headers = getAuthHeaders()
      const res = await fetch('/api/tasks?limit=200', { headers })
      if (!res.ok) return
      const data = await res.json()
      const tasks = data.tasks || []
      setInProgress(
        tasks.filter((t: { task_status: string }) => t.task_status === 'in_progress').length,
      )
      setCompleted(
        tasks.filter((t: { task_status: string }) => t.task_status === 'completed').length,
      )
      setFailed(tasks.filter((t: { task_status: string }) => t.task_status === 'failed').length)
      setWaiting(tasks.filter((t: { task_status: string }) => t.task_status === 'waiting').length)
    } catch {
      /* ignore */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetch_()
  }, [fetch_])

  return (
    <div className="px-2.5 py-2 border-b border-base-200 bg-base-300">
      <div className="flex items-center gap-1.5 flex-wrap">
        <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-xs font-medium border border-primary/25 bg-primary/10 text-primary shrink-0">
          <Loader2 size={11} className={inProgress > 0 ? 'animate-spin' : ''} />
          <span className="text-2xs opacity-85">进行中</span>
          <span className="font-bold text-xs font-mono">{inProgress}</span>
        </div>
        <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-xs font-medium border border-success/20 bg-success/[0.08] text-success shrink-0">
          <CheckCircle2 size={11} />
          <span className="text-2xs opacity-85">已完成</span>
          <span className="font-bold text-xs font-mono">{completed}</span>
        </div>
        {failed > 0 && (
          <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-xs font-medium border border-error/20 bg-error/[0.08] text-error shrink-0">
            <XCircle size={11} />
            <span className="font-bold text-xs font-mono">{failed}</span>
          </div>
        )}
        {waiting > 0 && (
          <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-xs font-medium border border-warning/20 bg-warning/[0.08] text-warning shrink-0">
            <Clock size={11} />
            <span className="font-bold text-xs font-mono">{waiting}</span>
          </div>
        )}
        <button
          type="button"
          className="ml-auto bg-transparent border-none text-on-surface-muted cursor-pointer p-0.5 rounded flex items-center transition-all duration-150 hover:text-on-surface hover:bg-base-200 disabled:opacity-40 disabled:cursor-not-allowed"
          onClick={fetch_}
          disabled={loading}
          title="Refresh"
        >
          <RefreshCw size={11} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>
    </div>
  )
}
