import { Check, ChevronDown, ChevronUp, Pencil, Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Profile } from '../../types'
import { getAuthHeaders } from '../../utils/auth'
import { ConfirmDialog } from './ConfirmDialog'
import { InlineInput } from './InlineInput'

interface Props {
  currentProfile: Profile | null
  onProfileChange: (profile: Profile) => void
}

export function ProfileSelector({ currentProfile, onProfileChange }: Props) {
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [newName, setNewName] = useState('')
  const [editName, setEditName] = useState('')
  const [loading, setLoading] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const fetchProfiles = useCallback(async () => {
    try {
      const headers = getAuthHeaders()
      const res = await fetch('/api/profiles', { headers })
      const data = await res.json()
      setProfiles(data.profiles || [])
      if (!currentProfile && data.profiles?.length > 0) {
        onProfileChange(data.profiles[0])
      }
    } catch (_err) {}
  }, [currentProfile, onProfileChange])

  useEffect(() => {
    fetchProfiles()
  }, [fetchProfiles])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false)
        setIsCreating(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    if (isCreating && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isCreating])

  const createProfile = async () => {
    if (!newName.trim() || loading) return
    setLoading(true)
    try {
      const name = newName.trim()
      const profile_key =
        name
          .toLowerCase()
          .replace(/[^a-z0-9]+/g, '-')
          .replace(/^-|-$/g, '') || `profile-${Date.now()}`
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }
      const res = await fetch('/api/profiles', {
        method: 'POST',
        headers,
        body: JSON.stringify({ name, profile_key }),
      })
      const data = await res.json()
      if (data.id) {
        const newProfile: Profile = {
          id: data.id,
          profile_key: data.profile_key,
          name: data.name,
          sort_order: profiles.length,
        }
        setProfiles([...profiles, newProfile])
        onProfileChange(newProfile)
        setNewName('')
        setIsCreating(false)
      }
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  const updateProfile = async () => {
    if (!currentProfile || !editName.trim() || loading) return
    setLoading(true)
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      }
      await fetch(`/api/profiles/${currentProfile.id}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({ name: editName.trim() }),
      })
      const updated = profiles.map((p) =>
        p.id === currentProfile.id ? { ...p, name: editName.trim() } : p,
      )
      setProfiles(updated)
      onProfileChange({ ...currentProfile, name: editName.trim() })
      setIsEditing(false)
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const deleteProfileClick = () => {
    if (!currentProfile || profiles.length <= 1 || loading) return
    setShowDeleteConfirm(true)
  }

  const deleteProfile = async () => {
    if (!currentProfile) return
    setShowDeleteConfirm(false)
    setLoading(true)
    try {
      const headers = getAuthHeaders()
      await fetch(`/api/profiles/${currentProfile.id}`, {
        method: 'DELETE',
        headers,
      })
      const remaining = profiles.filter((p) => p.id !== currentProfile.id)
      setProfiles(remaining)
      const first = remaining[0]
      if (first) onProfileChange(first)
      setIsEditing(false)
    } catch (_err) {
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex-1" ref={dropdownRef}>
      <button
        type="button"
        data-testid="profile-current"
        className="btn btn-ghost btn-sm flex items-center gap-2 flex-1 justify-between"
        onClick={() => setIsOpen(!isOpen)}
      >
        <span className="flex-1 font-medium text-sm text-left">
          {currentProfile?.name || 'Select Profile'}
        </span>
        <span className="text-base-content/50 flex items-center">
          {isOpen ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        </span>
      </button>

      {isOpen && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-base-200 border border-base-300 rounded-box shadow-md z-[100] overflow-hidden">
          <div className="max-h-[200px] overflow-y-auto">
            {profiles.map((profile) => (
              <button
                key={profile.id}
                type="button"
                className={`profile-item flex items-center justify-between px-3.5 py-2.5 cursor-pointer text-sm transition-colors duration-100 ${profile.id === currentProfile?.id ? 'bg-primary/10 text-primary active' : 'text-base-content hover:bg-base-300'}`}
                onClick={() => {
                  onProfileChange(profile)
                  setIsOpen(false)
                }}
              >
                <span>{profile.name}</span>
                {profile.id === currentProfile?.id && (
                  <Check size={14} className="text-primary text-xs" />
                )}
              </button>
            ))}
          </div>

          <div className="border-t border-base-300 p-2" data-testid="profile-actions">
            {isCreating ? (
              <InlineInput
                ref={inputRef}
                value={newName}
                onChange={setNewName}
                onSubmit={createProfile}
                onCancel={() => setIsCreating(false)}
                placeholder="Profile name..."
                loading={loading}
                name="profile-name"
                confirmTestId="btn-confirm"
                cancelTestId="btn-cancel"
              />
            ) : (
              <button
                type="button"
                className="btn btn-ghost btn-sm w-full border border-dashed border-base-300"
                onClick={() => setIsCreating(true)}
              >
                <Plus size={14} /> New Profile
              </button>
            )}
          </div>

          {currentProfile && (
            <div className="border-t border-base-300 p-2" data-testid="profile-edit-section">
              {isEditing ? (
                <InlineInput
                  value={editName}
                  onChange={setEditName}
                  onSubmit={updateProfile}
                  onCancel={() => setIsEditing(false)}
                  placeholder="Rename profile..."
                  loading={loading}
                  confirmTestId="edit-confirm"
                  cancelTestId="edit-cancel"
                />
              ) : (
                <div className="flex gap-2">
                  <button
                    type="button"
                    className="btn btn-ghost btn-xs flex-1"
                    onClick={() => {
                      setEditName(currentProfile.name)
                      setIsEditing(true)
                    }}
                  >
                    <Pencil size={12} className="mr-1" /> Edit
                  </button>
                  <button
                    type="button"
                    className="btn btn-ghost btn-xs flex-1 text-error hover:bg-error/10"
                    onClick={deleteProfileClick}
                    disabled={profiles.length <= 1}
                    title={profiles.length <= 1 ? 'Cannot delete last profile' : ''}
                  >
                    <Trash2 size={12} className="mr-1" /> Delete
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <ConfirmDialog
        open={showDeleteConfirm}
        title="Delete Profile"
        message={
          currentProfile ? `Delete profile "${currentProfile.name}"? This cannot be undone.` : ''
        }
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={deleteProfile}
        onCancel={() => setShowDeleteConfirm(false)}
      />
    </div>
  )
}
