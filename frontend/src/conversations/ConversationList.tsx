import { useMemo, useState } from 'react'
import { BUTTON_RESET } from '../shared/styles'
import type { AISession } from './types'
import { useConversations } from './useConversations'
import './conversations.css'

interface ConversationListProps {
  selectedSessionId: string | null
  onSelectSession: (session: AISession) => void
}

function groupByType(sessions: AISession[]): Map<string, AISession[]> {
  const groups = new Map<string, AISession[]>()
  for (const session of sessions) {
    const key = session.type === 'claude_code' ? 'Claude Code' : 'Codex'
    const existing = groups.get(key)
    if (existing) {
      groups.set(key, [...existing, session])
    } else {
      groups.set(key, [session])
    }
  }
  return groups
}

function formatRelativeTime(timestamp?: string): string {
  if (!timestamp) return ''
  try {
    const date = new Date(timestamp)
    if (Number.isNaN(date.getTime())) return ''
    const now = Date.now()
    const diffMs = now - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    if (diffMins < 1) return 'just now'
    if (diffMins < 60) return `${diffMins}m ago`
    const diffHours = Math.floor(diffMins / 60)
    if (diffHours < 24) return `${diffHours}h ago`
    const diffDays = Math.floor(diffHours / 24)
    return `${diffDays}d ago`
  } catch {
    return ''
  }
}

const TYPE_ORDER: Record<string, number> = {
  'Claude Code': 0,
  Codex: 1,
}

export function ConversationList({ selectedSessionId, onSelectSession }: ConversationListProps) {
  const [filterText, setFilterText] = useState('')
  const { sessions, loading, error, refetch } = useConversations(null)

  const filteredSessions = useMemo(() => {
    const lower = filterText.toLowerCase()
    if (!lower) return sessions
    return sessions.filter(
      (s) =>
        s.title.toLowerCase().includes(lower) ||
        s.model.toLowerCase().includes(lower) ||
        s.type.toLowerCase().includes(lower),
    )
  }, [sessions, filterText])

  const grouped = useMemo(() => groupByType(filteredSessions), [filteredSessions])
  const sortedGroups = useMemo(
    () =>
      [...grouped.entries()].sort((a, b) => (TYPE_ORDER[a[0]] ?? 99) - (TYPE_ORDER[b[0]] ?? 99)),
    [grouped],
  )

  return (
    <div className="conv-list-container">
      <div className="conv-list-header">
        <h3 className="conv-list-title">Conversations</h3>
        <div className="conv-list-filter-row">
          <input
            className="conv-list-filter-input"
            type="text"
            placeholder="Filter by title, model..."
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
          />
          <button
            type="button"
            className="conv-viewer-header-btn"
            onClick={refetch}
            disabled={loading}
            title="Refresh sessions"
          >
            <span className={loading ? 'conv-spinning' : ''}>&#8635;</span>
          </button>
        </div>
      </div>

      {error && <div className="conv-list-error">{error}</div>}

      <div className="conv-list-body">
        {loading && sessions.length === 0 && !error ? (
          <div className="conv-list-loading">
            <span className="conv-spinning">&#8635;</span> Loading...
          </div>
        ) : filteredSessions.length === 0 && !error ? (
          <div className="conv-list-empty">
            {filterText ? 'No matching conversations' : 'No conversations found'}
          </div>
        ) : (
          sortedGroups.map(([groupName, items]) => (
            <div key={groupName}>
              <div className="conv-group-header">
                {groupName}
                <span className="conv-group-count">{items.length}</span>
              </div>
              <div className="conv-group-list">
                {items.map((session) => (
                  <button
                    key={session.id}
                    type="button"
                    className={`conv-session-item${
                      selectedSessionId === session.id ? ' conv-session-item--selected' : ''
                    }`}
                    onClick={() => onSelectSession(session)}
                    style={BUTTON_RESET}
                  >
                    <span className="conv-session-title">{session.title || 'Untitled'}</span>
                    <div className="conv-session-meta">
                      <span className="conv-session-badge">{session.model}</span>
                      <span>{session.message_count} msgs</span>
                      <span>{formatRelativeTime(session.timestamp)}</span>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
