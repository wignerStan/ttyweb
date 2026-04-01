import { CheckCircle2, Clock, Loader2, RefreshCw, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { getAuthHeaders } from '../../utils/auth'
import './TaskStatBadges.css'

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
    <div className="task-stat-badges">
      <div className="task-stat-badges-row">
        <div className="task-stat-badge task-stat-badge--progress">
          <Loader2 size={11} className={inProgress > 0 ? 'task-stat-spin' : ''} />
          <span className="task-stat-label">进行中</span>
          <span className="task-stat-count">{inProgress}</span>
        </div>
        <div className="task-stat-badge task-stat-badge--done">
          <CheckCircle2 size={11} />
          <span className="task-stat-label">已完成</span>
          <span className="task-stat-count">{completed}</span>
        </div>
        {failed > 0 && (
          <div className="task-stat-badge task-stat-badge--failed">
            <XCircle size={11} />
            <span className="task-stat-count">{failed}</span>
          </div>
        )}
        {waiting > 0 && (
          <div className="task-stat-badge task-stat-badge--waiting">
            <Clock size={11} />
            <span className="task-stat-count">{waiting}</span>
          </div>
        )}
        <button
          type="button"
          className="task-stat-refresh"
          onClick={fetch_}
          disabled={loading}
          title="Refresh"
        >
          <RefreshCw size={11} className={loading ? 'task-stat-spin' : ''} />
        </button>
      </div>
    </div>
  )
}
