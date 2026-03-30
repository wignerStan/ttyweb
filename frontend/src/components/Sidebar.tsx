import type React from 'react'
import { useCallback, useEffect, useState } from 'react'

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

interface ApiResponse<T> {
  success: boolean
  data: T
  error?: string
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

  const fetchSessions = useCallback(async () => {
    try {
      const res = await fetch('/api/sessions')
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

  const fetchDetail = useCallback(
    async (name: string) => {
      if (details[name]) return
      try {
        const res = await fetch(`/api/sessions/${encodeURIComponent(name)}`)
        const json: ApiResponse<SessionDetail> = await res.json()
        if (json.success) {
          setDetails((prev) => ({ ...prev, [name]: json.data }))
        }
      } catch {
        // ignore
      }
    },
    [details],
  )

  useEffect(() => {
    fetchSessions()
    const interval = setInterval(fetchSessions, 3000)
    return () => clearInterval(interval)
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
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: newName.trim() }),
    })
    setNewName('')
    fetchSessions()
  }

  const handleKill = async (name: string) => {
    await fetch(`/api/sessions/${encodeURIComponent(name)}`, { method: 'DELETE' })
    if (expanded === name) setExpanded(null)
    fetchSessions()
  }

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h3 style={styles.title}>Sessions</h3>
        <div style={styles.createRow}>
          <input
            style={styles.input}
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            placeholder="new session"
          />
          <button type="button" style={styles.createBtn} onClick={handleCreate}>
            +
          </button>
        </div>
      </div>

      {error && <div style={styles.error}>{error}</div>}

      <div style={styles.list}>
        {sessions.map((s) => (
          <div key={s.name}>
            <button
              type="button"
              style={{
                background: 'none',
                border: 'none',
                padding: 0,
                font: 'inherit',
                color: 'inherit',
                cursor: 'pointer',
                width: '100%',
                textAlign: 'inherit',
                ...styles.sessionRow,
              }}
              onClick={() => handleToggle(s.name)}
            >
              <span style={styles.expandIcon}>{expanded === s.name ? '▼' : '▶'}</span>
              <span style={styles.sessionName} data-testid="session-name">
                {s.name}
              </span>
              {s.attached && <span style={styles.badge}>A</span>}
              <button
                type="button"
                style={styles.killBtn}
                onClick={(e) => {
                  e.stopPropagation()
                  handleKill(s.name)
                }}
                title="Kill session"
              >
                ×
              </button>
            </button>

            {expanded === s.name && details[s.name] && (
              <div style={styles.paneList}>
                {details[s.name]?.panes.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    style={{
                      background: 'none',
                      border: 'none',
                      padding: 0,
                      font: 'inherit',
                      color: 'inherit',
                      cursor: 'pointer',
                      width: '100%',
                      textAlign: 'inherit',
                      ...styles.paneRow,
                    }}
                    onClick={() => onSelect(s.name, p.id)}
                  >
                    <span style={styles.paneId}>{p.id}</span>
                    <span style={styles.paneCmd}>{p.current_command}</span>
                    {!p.running && <span style={styles.deadBadge}>dead</span>}
                  </button>
                ))}
                <button
                  type="button"
                  style={{
                    background: 'none',
                    border: 'none',
                    padding: 0,
                    font: 'inherit',
                    color: 'inherit',
                    cursor: 'pointer',
                    width: '100%',
                    textAlign: 'inherit',
                    ...styles.connectAll,
                  }}
                  onClick={() => onSelect(s.name)}
                >
                  Connect to session
                </button>
              </div>
            )}
          </div>
        ))}
        {sessions.length === 0 && !error && <div style={styles.empty}>No sessions found</div>}
      </div>
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
  },
  header: {
    padding: '12px',
    borderBottom: '1px solid #24283b',
  },
  title: {
    margin: '0 0 8px 0',
    fontSize: '14px',
    fontWeight: 600,
    color: '#c0caf5',
  },
  createRow: {
    display: 'flex',
    gap: '4px',
  },
  input: {
    flex: 1,
    background: '#1a1b26',
    border: '1px solid #24283b',
    color: '#c0caf5',
    padding: '4px 8px',
    fontSize: '12px',
    borderRadius: '2px',
    outline: 'none',
  },
  createBtn: {
    background: '#24283b',
    border: '1px solid #3b4261',
    color: '#c0caf5',
    padding: '4px 10px',
    cursor: 'pointer',
    borderRadius: '2px',
    fontSize: '14px',
    fontWeight: 'bold',
  },
  error: {
    padding: '8px 12px',
    color: '#f7768e',
    fontSize: '12px',
  },
  list: {
    flex: 1,
    overflow: 'auto',
    padding: '4px 0',
  },
  sessionRow: {
    display: 'flex',
    alignItems: 'center',
    padding: '6px 12px',
    cursor: 'pointer',
    gap: '6px',
  },
  expandIcon: {
    fontSize: '10px',
    color: '#565f89',
    width: '12px',
    textAlign: 'center' as const,
  },
  sessionName: {
    flex: 1,
    fontSize: '13px',
  },
  badge: {
    fontSize: '10px',
    background: '#3b4261',
    color: '#7aa2f7',
    padding: '1px 5px',
    borderRadius: '2px',
  },
  killBtn: {
    background: 'none',
    border: 'none',
    color: '#565f89',
    cursor: 'pointer',
    fontSize: '16px',
    padding: '0',
    lineHeight: '1',
  },
  paneList: {
    paddingLeft: '24px',
  },
  paneRow: {
    display: 'flex',
    alignItems: 'center',
    padding: '4px 12px',
    cursor: 'pointer',
    gap: '8px',
    fontSize: '12px',
  },
  paneId: {
    color: '#7aa2f7',
    fontFamily: 'monospace',
    minWidth: '30px',
  },
  paneCmd: {
    flex: 1,
    color: '#9ece6a',
  },
  deadBadge: {
    fontSize: '10px',
    background: '#414868',
    color: '#f7768e',
    padding: '1px 4px',
    borderRadius: '2px',
  },
  connectAll: {
    padding: '4px 12px',
    cursor: 'pointer',
    color: '#7aa2f7',
    fontSize: '11px',
    fontStyle: 'italic',
  },
  empty: {
    padding: '12px',
    color: '#565f89',
    fontSize: '12px',
    textAlign: 'center' as const,
  },
}
