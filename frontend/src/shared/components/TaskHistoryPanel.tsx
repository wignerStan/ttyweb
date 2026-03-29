import {
  CheckCheck,
  CheckCircle2,
  Clock,
  History,
  Loader2,
  RefreshCw,
  Terminal,
  X,
  XCircle,
} from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { getAuthHeader } from '../../utils/auth'
import './TaskHistoryPanel.css'

interface Conversation {
  id: number
  conversation_id: string
  pane_key: string
  user_message: string
  assistant_message: string | null
  conv_status: string
  started_at: number
  completed_at: number | null
}

interface TaskHistoryPanelProps {
  paneKey: string | null
  onClose: () => void
  embedded?: boolean
  onStatusChange?: () => void
}

function StatusIcon({ status }: { status: string }) {
  if (status === 'in_progress')
    return <Loader2 size={14} className="task-history-spin" style={{ color: 'var(--blue-400)' }} />
  if (status === 'completed')
    return <CheckCircle2 size={14} style={{ color: 'var(--green-500)' }} />
  if (status === 'failed') return <XCircle size={14} style={{ color: 'var(--red-500)' }} />
  if (status === 'waiting') return <Clock size={14} style={{ color: 'var(--yellow-500)' }} />
  return <Clock size={14} style={{ color: 'var(--zinc-500)' }} />
}

function formatTime(ts: number): string {
  return new Date(ts * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatDuration(start: number, end: number | null): string {
  const secs = (end ?? Math.floor(Date.now() / 1000)) - start
  if (secs < 60) return `${secs}s`
  if (secs < 3600) return `${Math.floor(secs / 60)}m ${secs % 60}s`
  return `${Math.floor(secs / 3600)}h ${Math.floor((secs % 3600) / 60)}m`
}

export function TaskHistoryPanel({
  paneKey,
  onClose,
  embedded,
  onStatusChange,
}: TaskHistoryPanelProps) {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [loading, setLoading] = useState(false)

  const fetchHistory = useCallback(async () => {
    if (!paneKey) return
    setLoading(true)
    try {
      const urlKey = paneKey.replace(/:/g, '/')
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers.Authorization = auth
      const res = await fetch(`/api/tasks/events/${encodeURIComponent(urlKey)}?limit=30`, {
        headers,
      })
      if (!res.ok) throw new Error('Failed to fetch')
      const data = await res.json()
      setConversations(data.conversations || [])
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }, [paneKey])

  useEffect(() => {
    fetchHistory()
  }, [fetchHistory])

  const markComplete = useCallback(
    async (id: number) => {
      // Optimistically update local state
      setConversations((prev) =>
        prev.map((c) =>
          c.id === id
            ? { ...c, conv_status: 'completed', completed_at: Math.floor(Date.now() / 1000) }
            : c,
        ),
      )
      try {
        const auth = getAuthHeader()
        const headers: Record<string, string> = {}
        if (auth) headers.Authorization = auth
        await fetch(`/api/tasks/conv/${id}/complete`, {
          method: 'PATCH',
          headers,
        })
        onStatusChange?.() // tell parent to refresh TmuxTree status badges
      } catch (_err) {
        fetchHistory()
      }
    },
    [fetchHistory, onStatusChange],
  )

  const displayName = paneKey ? paneKey.split(':').slice(0, 2).join(':') : '—'

  const inner = (
    <>
      {/* Header */}
      <div className="task-history-header">
        <div className="task-history-header-left">
          <History size={15} className="task-history-header-icon" />
          <span className="task-history-title">{paneKey ? displayName : '任务历史'}</span>
        </div>
        <div className="task-history-header-right">
          <button
            className="task-history-btn"
            onClick={fetchHistory}
            disabled={loading}
            title="Refresh"
          >
            <RefreshCw size={13} className={loading ? 'task-history-spin' : ''} />
          </button>
          {!embedded && (
            <button className="task-history-btn" onClick={onClose} title="Close">
              <X size={15} />
            </button>
          )}
        </div>
      </div>

      {/* Pane context */}
      {paneKey && (
        <div className="task-history-context">
          <Terminal size={12} style={{ color: 'var(--zinc-500)', flexShrink: 0 }} />
          <span className="task-history-pane-name">{displayName}</span>
        </div>
      )}

      {/* List */}
      <div className="task-history-list">
        {!paneKey ? (
          <div className="task-history-empty">点击 pane 的状态图标查看任务历史</div>
        ) : loading && conversations.length === 0 ? (
          <div className="task-history-loading">
            <Loader2 size={20} className="task-history-spin" />
            <span>Loading...</span>
          </div>
        ) : conversations.length === 0 ? (
          <div className="task-history-empty">该 pane 暂无任务历史</div>
        ) : (
          conversations.map((conv) => (
            <div key={conv.id} className={`task-history-item status-${conv.conv_status}`}>
              <div className="task-history-item-top">
                <StatusIcon status={conv.conv_status} />
                <span className="task-history-item-msg">{conv.user_message || 'Untitled'}</span>
                <span className="task-history-item-time">{formatTime(conv.started_at)}</span>
                {conv.conv_status === 'in_progress' && (
                  <button
                    className="task-history-complete-btn"
                    onClick={() => markComplete(conv.id)}
                    title="标记为已完成"
                  >
                    <CheckCheck size={12} />
                  </button>
                )}
              </div>
              {conv.assistant_message && (
                <div className="task-history-item-reply">{conv.assistant_message}</div>
              )}
              <div className="task-history-item-meta">
                <span className="task-history-badge">{conv.conv_status}</span>
                <span className="task-history-badge">
                  {formatDuration(conv.started_at, conv.completed_at)}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </>
  )

  if (embedded) {
    return <div className="task-history-panel task-history-embedded">{inner}</div>
  }

  return <aside className="task-history-panel">{inner}</aside>
}
