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
    <aside
      className={`mobile-drawer fixed left-0 top-0 bottom-0 z-[var(--z-drawer)] flex w-[280px] flex-col overflow-hidden bg-base-200 transition-transform duration-[250ms] ease-out ${open ? 'open translate-x-0' : '-translate-x-full'}`}
    >
      <div className="flex h-12 shrink-0 items-center justify-between border-b border-base-200 px-3">
        <div className="mobile-drawer-actions ml-0 flex flex-1 gap-1">
          <button
            className="btn btn-ghost btn-sm btn-circle min-h-[44px] min-w-[44px] text-base-content/45"
            onClick={() => setShowGroupManager(!showGroupManager)}
            type="button"
            title="Manage groups"
          >
            <Settings size={18} />
          </button>
          <button
            className="btn btn-ghost btn-sm btn-circle min-h-[44px] min-w-[44px] text-base-content/45"
            onClick={onRefresh}
            type="button"
            title="Refresh"
          >
            <RefreshCw size={18} />
          </button>
          <button
            className="btn btn-ghost btn-sm btn-circle min-h-[44px] min-w-[44px] text-base-content/45"
            onClick={onLogout}
            type="button"
            title="Sign out"
          >
            <LogOut size={16} />
          </button>
          <button
            className="btn btn-ghost btn-sm btn-circle min-h-[44px] min-w-[44px] text-base-content/45"
            onClick={onClose}
            type="button"
            title="Close"
          >
            <X size={18} />
          </button>
        </div>
      </div>

      <div className="mobile-drawer-content flex-1 overflow-y-auto touch-scroll">
        <div className="mobile-drawer-profile flex items-center gap-2 border-b border-base-200 px-3 py-2">
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

        <div className="mobile-drawer-scrollable flex flex-1 flex-col overflow-y-auto">
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
