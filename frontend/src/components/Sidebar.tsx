import { useCallback, useEffect, useRef, useState } from 'react'
import { ConfirmDialog } from '../shared/components/ConfirmDialog'
import type { ApiResponse } from '../types'
import { getAuthHeaders } from '../utils/auth'

interface Session {
  name: string
  windows: number
  attached: boolean
}

interface Pane {
  id: string
  title: string
  current_command: string
  running: boolean
}

interface SessionDetail {
  name: string
  windows: number
  attached: boolean
  panes: Pane[]
}

interface SidebarProps {
  onSelect: (session: string, pane?: string) => void
}

export function Sidebar({ onSelect }: SidebarProps) {
  const [sessions, setSessions] = useState<Session[]>([])
  const [expanded, setExpanded] = useState<string | null>(null)
  const [details, setDetails] = useState<Record<string, SessionDetail>>({})
  const [newName, setNewName] = useState('')
  const [error, setError] = useState('')
  const [killTarget, setKillTarget] = useState<string | null>(null)

  const detailsRef = useRef(details)
  detailsRef.current = details

  const fetchSessions = useCallback(async () => {
    try {
      const res = await fetch('/api/sessions', { headers: getAuthHeaders() })
      const json: ApiResponse<Session[]> = await res.json()
      if (json.success) {
        setSessions(json.data)
        setError('')
      } else {
        setError(json.error ?? 'failed to list sessions')
      }
    } catch {
      setError('failed to connect to server')
    }
  }, [])

  const fetchDetail = useCallback(async (name: string) => {
    if (detailsRef.current[name]) return
    try {
      const res = await fetch(`/api/sessions/${encodeURIComponent(name)}`, {
        headers: getAuthHeaders(),
      })
      const json: ApiResponse<SessionDetail> = await res.json()
      if (json.success) {
        setDetails((prev) => ({ ...prev, [name]: json.data }))
      }
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    fetchSessions()
    const interval = setInterval(fetchSessions, 3000)
    const handleVisibility = () => {
      if (document.visibilityState === 'visible') {
        fetchSessions()
      }
    }
    document.addEventListener('visibilitychange', handleVisibility)
    return () => {
      clearInterval(interval)
      document.removeEventListener('visibilitychange', handleVisibility)
    }
  }, [fetchSessions])

  const handleToggle = (name: string) => {
    if (expanded === name) {
      setExpanded(null)
    } else {
      setExpanded(name)
      fetchDetail(name)
    }
  }

  const handleCreate = async () => {
    if (!newName.trim()) return
    await fetch('/api/sessions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ name: newName.trim() }),
    })
    setNewName('')
    fetchSessions()
  }

  const handleKill = (name: string) => {
    setKillTarget(name)
  }

  const confirmKill = async () => {
    if (!killTarget) return
    await fetch(`/api/sessions/${encodeURIComponent(killTarget)}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    })
    if (expanded === killTarget) setExpanded(null)
    fetchSessions()
    setKillTarget(null)
  }

  return (
    <div className="flex flex-col h-full">
      <div className="p-3 border-b border-base-300">
        <h3 className="m-0 mb-2 text-sm font-semibold text-base-content">Sessions</h3>
        <div className="flex gap-1">
          <input
            className="input input-bordered input-xs flex-1"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            placeholder="New session..."
          />
          <button
            type="button"
            className="btn btn-ghost btn-xs font-bold"
            onClick={handleCreate}
            aria-label="New session"
          >
            +
          </button>
        </div>
      </div>

      {error && <div className="px-3 py-2 text-error text-xs">{error}</div>}

      <div className="flex-1 overflow-auto py-1">
        {sessions.map((s) => (
          <div key={s.name}>
            <button
              type="button"
              className="btn-reset flex items-center px-3 py-1.5 cursor-pointer gap-1.5"
              onClick={() => handleToggle(s.name)}
            >
              <span className="text-[10px] text-base-content/40 w-3 text-center">
                {expanded === s.name ? '▼' : '▶'}
              </span>
              <span className="flex-1 text-[13px]" data-testid="session-name">
                {s.name}
              </span>
              {s.attached && <span className="badge badge-xs badge-ghost text-primary">A</span>}
              <button
                type="button"
                className="btn-reset text-base-content/40 text-base cursor-pointer hover:text-error"
                onClick={(e) => {
                  e.stopPropagation()
                  handleKill(s.name)
                }}
                title="Kill session"
                aria-label="Kill session"
              >
                ×
              </button>
            </button>

            {expanded === s.name && details[s.name] && (
              <div className="pl-6">
                {details[s.name]?.panes.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    className="btn-reset flex items-center px-3 py-1 cursor-pointer gap-2 text-xs"
                    onClick={() => onSelect(s.name, p.id)}
                  >
                    <span className="text-primary font-mono min-w-[30px]">{p.id}</span>
                    <span className="flex-1 text-success">{p.current_command}</span>
                    {!p.running && (
                      <span className="text-[10px] bg-base-300 text-error px-1 py-px rounded-sm">
                        dead
                      </span>
                    )}
                  </button>
                ))}
                <button
                  type="button"
                  className="btn-reset px-3 py-1 cursor-pointer text-primary text-[11px] italic"
                  onClick={() => onSelect(s.name)}
                >
                  Connect to session
                </button>
              </div>
            )}
          </div>
        ))}
        {sessions.length === 0 && !error && (
          <div className="p-3 text-base-content/40 text-xs text-center">No sessions found</div>
        )}
      </div>
      <ConfirmDialog
        open={killTarget !== null}
        title="Kill Session"
        message={`Are you sure you want to kill session "${killTarget}"? This cannot be undone.`}
        confirmLabel="Kill"
        variant="destructive"
        onConfirm={confirmKill}
        onCancel={() => setKillTarget(null)}
      />
    </div>
  )
}
