import { ArrowRight, Pencil, Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { SessionGroup, TmuxSession } from '../../types'
import { getAuthHeaders } from '../../utils/auth'
import { ConfirmDialog } from './ConfirmDialog'
import { InlineInput } from './InlineInput'

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
      const headers = getAuthHeaders()
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
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }
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
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }
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

  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)

  const deleteGroup = async (id: number) => {
    if (loading) return
    setDeleteTarget(id)
  }

  const handleDeleteConfirm = async () => {
    if (deleteTarget === null) return
    const group = groups.find((g) => g.id === deleteTarget)
    if (!group) return
    setLoading(true)
    try {
      const headers = getAuthHeaders()
      await fetch(`/api/groups/${deleteTarget}`, {
        method: 'DELETE',
        headers,
      })
      setGroups(groups.filter((g) => g.id !== deleteTarget))
      onGroupsChanged()
    } catch (_err) {
    } finally {
      setLoading(false)
      setDeleteTarget(null)
    }
  }

  const assignSessionToGroup = async (sessionName: string, groupId: number | null) => {
    setLoading(true)
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }
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
          <InlineInput
            ref={inputRef}
            value={newGroupName}
            onChange={setNewGroupName}
            onSubmit={createGroup}
            onCancel={() => setIsCreating(false)}
            placeholder="Group name..."
            loading={loading}
            name="group-name"
            inputClassName="group-input"
            confirmClassName="btn-sm btn-confirm"
            cancelClassName="btn-sm btn-cancel"
            iconSize={12}
            className="flex gap-1.5 w-full"
          />
        </div>
      )}

      <div className="group-list">
        {groups.length === 0 && !isCreating && <div className="group-empty">No groups yet</div>}
        {groups.map((group) => (
          <div key={group.id} className="group-item">
            {editingId === group.id ? (
              <div className="group-edit-row">
                <InlineInput
                  value={editName}
                  onChange={setEditName}
                  onSubmit={() => updateGroup(group.id)}
                  onCancel={() => setEditingId(null)}
                  loading={loading}
                  inputClassName="group-input"
                  confirmClassName="btn-sm btn-confirm"
                  cancelClassName="btn-sm btn-cancel"
                  iconSize={12}
                  className="flex gap-1.5 w-full"
                />
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

      <ConfirmDialog
        open={deleteTarget !== null}
        title="Delete Group"
        message={
          deleteTarget !== null
            ? `Delete group "${groups.find((g) => g.id === deleteTarget)?.group_name}"? This cannot be undone.`
            : ''
        }
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
