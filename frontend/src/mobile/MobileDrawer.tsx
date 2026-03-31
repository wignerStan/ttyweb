import { LogOut, RefreshCw, Settings, X } from 'lucide-react'
import { useState } from 'react'
import { GroupManager } from '../shared/components/GroupManager'
import { ProfileSelector } from '../shared/components/ProfileSelector'
import { TaskStatBadges } from '../shared/components/TaskStatBadges'
import { TmuxTree } from '../shared/components/TmuxTree'
import type { Profile, SessionGroup, TmuxSession } from '../types'

interface Props {
  open: boolean
  sessions: TmuxSession[]
  currentProfile: Profile | null
  groups: SessionGroup[]
  statusRefreshToken?: number
  onProfileChange: (profile: Profile) => void
  onGroupsChanged: () => void
  onSelectPane: (paneId: string, paneName: string) => void
  onPaneStatusClick?: (paneKey: string) => void
  onClose: () => void
  onRefresh: () => void
  onLogout: () => void
}

export function MobileDrawer({
  open,
  sessions,
  currentProfile,
  groups,
  statusRefreshToken,
  onProfileChange,
  onGroupsChanged,
  onSelectPane,
  onPaneStatusClick,
  onClose,
  onRefresh,
  onLogout,
}: Props) {
  const [showGroupManager, setShowGroupManager] = useState(false)

  const handleSelectPane = (paneId: string, paneName: string) => {
    onSelectPane(paneId, paneName)
    onClose()
  }

  const handlePaneStatusClick = (paneKey: string) => {
    onPaneStatusClick?.(paneKey)
    onClose() // close drawer, right panel will open
  }

  return (
    <aside className={`mobile-drawer ${open ? 'open' : ''}`}>
      <div className="mobile-drawer-header">
        <div className="mobile-drawer-actions" style={{ marginLeft: 0, flex: 1 }}>
          <button
            className="mobile-drawer-btn"
            onClick={() => setShowGroupManager(!showGroupManager)}
            type="button"
            title="Manage groups"
            aria-label="Settings"
          >
            <Settings size={18} />
          </button>
          <button
            className="mobile-drawer-btn"
            onClick={onRefresh}
            type="button"
            title="Refresh"
            aria-label="Refresh sessions"
          >
            <RefreshCw size={18} />
          </button>
          <button
            className="mobile-drawer-btn"
            onClick={onLogout}
            type="button"
            title="Sign out"
            aria-label="Logout"
          >
            <LogOut size={16} />
          </button>
          <button
            className="mobile-drawer-btn"
            onClick={onClose}
            type="button"
            title="Close"
            aria-label="Close drawer"
          >
            <X size={18} />
          </button>
        </div>
      </div>

      <div className="mobile-drawer-content">
        <div className="mobile-drawer-profile">
          <ProfileSelector currentProfile={currentProfile} onProfileChange={onProfileChange} />
        </div>

        {/* Task stat badges — compact, always visible */}
        <TaskStatBadges refreshToken={statusRefreshToken} />

        {showGroupManager && currentProfile && (
          <GroupManager
            profileKey={currentProfile.profile_key}
            sessions={sessions}
            onGroupsChanged={onGroupsChanged}
          />
        )}

        <div className="mobile-drawer-scrollable">
          <TmuxTree
            sessions={sessions}
            groups={groups}
            profileId={currentProfile?.id}
            profileKey={currentProfile?.profile_key}
            onSelectPane={handleSelectPane}
            onRefresh={onRefresh}
            onOrderChange={onGroupsChanged}
            onPaneStatusClick={handlePaneStatusClick}
            statusRefreshToken={statusRefreshToken}
            defaultExpanded={false}
          />
        </div>
      </div>
    </aside>
  )
}
