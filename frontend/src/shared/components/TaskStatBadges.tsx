import { CheckCircle2, Clock, Loader2, RefreshCw, XCircle } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { getAuthHeader } from '../../utils/auth'

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
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers.Authorization = auth
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
    <div className="px-2.5 py-2 border-b border-[var(--zinc-800)] bg-[var(--zinc-900)]">
      <div className="flex items-center gap-1.5 flex-wrap">
        <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-[11px] font-medium border border-blue-500/25 bg-blue-500/10 text-[var(--blue-400)] shrink-0">
          <Loader2 size={11} className={inProgress > 0 ? 'animate-spin' : ''} />
          <span className="text-[10px] opacity-85">进行中</span>
          <span className="font-bold text-xs font-mono">{inProgress}</span>
        </div>
        <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-[11px] font-medium border border-green-500/20 bg-green-500/[0.08] text-[var(--green-400)] shrink-0">
          <CheckCircle2 size={11} />
          <span className="text-[10px] opacity-85">已完成</span>
          <span className="font-bold text-xs font-mono">{completed}</span>
        </div>
        {failed > 0 && (
          <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-[11px] font-medium border border-red-500/20 bg-red-500/[0.08] text-[var(--red-400)] shrink-0">
            <XCircle size={11} />
            <span className="font-bold text-xs font-mono">{failed}</span>
          </div>
        )}
        {waiting > 0 && (
          <div className="flex items-center gap-1 py-0.5 px-2 rounded-full text-[11px] font-medium border border-yellow-500/20 bg-yellow-500/[0.08] text-[var(--yellow-400)] shrink-0">
            <Clock size={11} />
            <span className="font-bold text-xs font-mono">{waiting}</span>
          </div>
        )}
        <button
          type="button"
          className="ml-auto bg-transparent border-none text-[var(--zinc-600)] cursor-pointer p-0.5 rounded flex items-center transition-all duration-150 hover:text-[var(--zinc-300)] hover:bg-[var(--zinc-800)] disabled:opacity-40 disabled:cursor-not-allowed"
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
