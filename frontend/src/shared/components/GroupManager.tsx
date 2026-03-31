import { ArrowRight, Check, Pencil, Plus, Trash2, X } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionGroup, TmuxSession } from '../../types'
import { getAuthHeader } from '../../utils/auth'
import './GroupManager.css'

interface Props {
  profileKey: string
  sessions: TmuxSession[]
  onGroupsChanged: () => void
}

export function GroupManager({ profileKey, sessions, onGroupsChanged }: Props) {
  const [groups, setGroups] = useState<SessionGroup[]>([])
  const [isCreating, setIsCreating] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [assigningSession, setAssigningSession] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const fetchGroups = useCallback(async () => {
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers.Authorization = auth
      const res = await fetch(`/api/groups?profile_key=${encodeURIComponent(profileKey)}`, {
        headers,
      })
      const data = await res.json()
      setGroups(data.groups || [])
    } catch (_err) {}
  }, [profileKey])

  useEffect(() => {
    if (profileKey) fetchGroups()
  }, [profileKey, fetchGroups])

  useEffect(() => {
    if (isCreating && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isCreating])

  const createGroup = async () => {
    if (!newGroupName.trim() || loading) return
    setLoading(true)
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (auth) headers.Authorization = auth
      const res = await fetch('/api/groups', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          profile_key: profileKey,
          group_name: newGroupName.trim(),
        }),
      })
      const data = await res.json()
      if (data.id) {
        const newGroup: SessionGroup = {
          id: data.id,
          group_name: data.group_name,
          sort_order: groups.length,
          session_count: 0,
        }
        setGroups([...groups, newGroup])
        setNewGroupName('')
        setIsCreating(false)
        onGroupsChanged()
      }
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  const updateGroup = async (id: number) => {
    if (!editName.trim() || loading) return
    setLoading(true)
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (auth) headers.Authorization = auth
      await fetch(`/api/groups/${id}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ group_name: editName.trim() }),
      })
      setGroups(groups.map((g) => (g.id === id ? { ...g, group_name: editName.trim() } : g)))
      setEditingId(null)
      onGroupsChanged()
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  const deleteGroup = async (id: number) => {
    const group = groups.find((g) => g.id === id)
    if (!group || loading) return
    if (!confirm(`Delete group "${group.group_name}"?`)) return
    setLoading(true)
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers.Authorization = auth
      await fetch(`/api/groups/${id}`, {
        method: 'DELETE',
        headers,
      })
      setGroups(groups.filter((g) => g.id !== id))
      onGroupsChanged()
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  const assignSessionToGroup = async (sessionName: string, groupId: number | null) => {
    setLoading(true)
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (auth) headers.Authorization = auth
      await fetch(`/api/sessions/${encodeURIComponent(sessionName)}/group`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ profile_key: profileKey, group_id: groupId }),
      })
      setAssigningSession(null)
      fetchGroups()
      onGroupsChanged()
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent, action: () => void) => {
    if (e.key === 'Enter') action()
    if (e.key === 'Escape') {
      setIsCreating(false)
      setEditingId(null)
      setAssigningSession(null)
    }
  }

  return (
    <div className="group-manager">
      <div className="group-header">
        <span className="group-title">Groups</span>
        <button
          type="button"
          className="group-add-btn"
          onClick={() => setIsCreating(true)}
          title="Create group"
          aria-label="Add group"
        >
          <Plus size={14} />
        </button>
      </div>

      {isCreating && (
        <div className="group-create-row">
          <input
            ref={inputRef}
            type="text"
            value={newGroupName}
            onChange={(e) => setNewGroupName(e.target.value)}
            onKeyDown={(e) => handleKeyDown(e, createGroup)}
            placeholder="Group name..."
            className="group-input"
            disabled={loading}
          />
          <button
            type="button"
            onClick={createGroup}
            disabled={loading || !newGroupName.trim()}
            className="btn-sm btn-confirm"
          >
            <Check size={12} />
          </button>
          <button type="button" onClick={() => setIsCreating(false)} className="btn-sm btn-cancel">
            <X size={12} />
          </button>
        </div>
      )}

      <div className="group-list">
        {groups.length === 0 && !isCreating && <div className="group-empty">No groups yet</div>}
        {groups.map((group) => (
          <div key={group.id} className="group-item">
            {editingId === group.id ? (
              <div className="group-edit-row">
                <input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  onKeyDown={(e) => handleKeyDown(e, () => updateGroup(group.id))}
                  className="group-input"
                  disabled={loading}
                />
                <button
                  type="button"
                  onClick={() => updateGroup(group.id)}
                  disabled={loading || !editName.trim()}
                  className="btn-sm btn-confirm"
                >
                  <Check size={12} />
                </button>
                <button
                  type="button"
                  onClick={() => setEditingId(null)}
                  className="btn-sm btn-cancel"
                >
                  <X size={12} />
                </button>
              </div>
            ) : (
              <div className="group-row">
                <span className="group-name">{group.group_name}</span>
                <span className="group-count">{group.session_count}</span>
                <div className="group-actions">
                  <button
                    type="button"
                    className="btn-icon"
                    onClick={() => {
                      setEditingId(group.id)
                      setEditName(group.group_name)
                    }}
                    title="Rename"
                    aria-label="Rename group"
                  >
                    <Pencil size={12} />
                  </button>
                  <button
                    type="button"
                    className="btn-icon btn-danger"
                    onClick={() => deleteGroup(group.id)}
                    title="Delete"
                    aria-label="Delete group"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="session-assign-section">
        <div className="section-label">Assign Sessions</div>
        <div className="session-list">
          {sessions.map((session) => (
            <div key={session.sessionId} className="session-row">
              <span className="session-name">{session.sessionName}</span>
              {assigningSession === session.sessionName ? (
                <select
                  className="group-select"
                  onChange={(e) => {
                    const val = e.target.value
                    assignSessionToGroup(session.sessionName, val ? parseInt(val, 10) : null)
                  }}
                  onBlur={() => setAssigningSession(null)}
                >
                  <option value="">Ungrouped</option>
                  {groups.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.group_name}
                    </option>
                  ))}
                </select>
              ) : (
                <button
                  type="button"
                  className="btn-assign"
                  onClick={() => setAssigningSession(session.sessionName)}
                  title="Assign to group"
                >
                  <ArrowRight size={14} />
                </button>
              )}
            </div>
          ))}
          {sessions.length === 0 && <div className="session-empty">No sessions available</div>}
        </div>
      </div>
    </div>
  )
}
