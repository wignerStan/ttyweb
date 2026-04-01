import { History, Menu, ScrollText, X } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import useShakeDetect from '../hooks/useShakeDetect'
import useVisualViewport from '../hooks/useVisualViewport'
import { ImperialStudyPanel } from '../shared/components/imperial-study/components/ImperialStudyPanel'
import { LoginModal } from '../shared/components/LoginModal'
import { TaskHistoryPanel } from '../shared/components/TaskHistoryPanel'
import type { VoiceInputHandle } from '../shared/components/VoiceInput'
import type { OpenTab, Profile, SessionGroup, TmuxSession } from '../types'
import { checkAuth, getAuthHeaders, logout } from '../utils/auth'
import { MobileDrawer } from './MobileDrawer'
import { MobileTerminal } from './MobileTerminal'

interface MobileTab extends OpenTab {
  session: string
}

function getAllPaneIds(sessions: TmuxSession[]): Set<string> {
  const ids = new Set<string>()
  for (const s of sessions) {
    for (const w of s.windows) {
      for (const p of w.panes) {
        ids.add(p.paneId)
      }
    }
  }
  return ids
}

function getPaneKey(sessions: TmuxSession[], paneId: string): string | null {
  for (const s of sessions) {
    for (let wi = 0; wi < s.windows.length; wi++) {
      const w = s.windows[wi]
      if (!w) continue
      for (let pi = 0; pi < w.panes.length; pi++) {
        if (w.panes[pi]?.paneId === paneId) {
          return `${s.sessionName}:${w.windowIndex}:${w.panes[pi]?.paneId}`
        }
      }
    }
  }
  return null
}

function getSessionForPane(sessions: TmuxSession[], paneId: string): string | null {
  for (const s of sessions) {
    for (const w of s.windows) {
      for (const p of w.panes) {
        if (p.paneId === paneId) {
          return s.sessionName
        }
      }
    }
  }
  return null
}

function loadTabs(): MobileTab[] {
  try {
    const raw = localStorage.getItem('mobile-openTabs')
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function saveTabs(tabs: MobileTab[]) {
  localStorage.setItem('mobile-openTabs', JSON.stringify(tabs))
}

function loadActiveTabId(): string | null {
  return localStorage.getItem('mobile-activeTabId') || null
}

function saveActiveTabId(id: string | null) {
  if (id) {
    localStorage.setItem('mobile-activeTabId', id)
  } else {
    localStorage.removeItem('mobile-activeTabId')
  }
}

export default function MobileApp() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean | null>(null)
  const [sessions, setSessions] = useState<TmuxSession[]>([])
  const [tabs, setTabs] = useState<MobileTab[]>(loadTabs)
  const [activeTabId, setActiveTabId] = useState<string | null>(loadActiveTabId)
  const [prevActiveTabId, setPrevActiveTabId] = useState<string | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [rightPanelOpen, setRightPanelOpen] = useState(false)
  const [taskHistoryPaneKey, setTaskHistoryPaneKey] = useState<string | null>(null)
  const [statusRefreshToken, setStatusRefreshToken] = useState(0)
  const [imperialOpen, setImperialOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fontSize, setFontSize] = useState(() => {
    const saved = localStorage.getItem('terminal-font-size')
    return saved ? parseFloat(saved) : 10
  })

  const [currentProfile, setCurrentProfile] = useState<Profile | null>(null)
  const [groups, setGroups] = useState<SessionGroup[]>([])

  const voiceRef = useRef<VoiceInputHandle>(null)

  useEffect(() => {
    saveTabs(tabs)
  }, [tabs])
  useEffect(() => {
    saveActiveTabId(activeTabId)
  }, [activeTabId])
  useEffect(() => {
    setTaskHistoryPaneKey(null)
  }, [])

  useEffect(() => {
    setPrevActiveTabId(activeTabId)
  }, [activeTabId])

  const liveTabIds = useMemo(() => {
    const ids = new Set<string>()
    if (activeTabId) ids.add(activeTabId)
    if (prevActiveTabId && prevActiveTabId !== activeTabId) ids.add(prevActiveTabId)
    return ids
  }, [activeTabId, prevActiveTabId])

  useVisualViewport()

  useShakeDetect(
    () => {
      voiceRef.current?.toggle()
    },
    { enabled: tabs.length > 0 },
  )

  const handleFontSizeChange = useCallback((size: number) => {
    setFontSize(size)
    localStorage.setItem('terminal-font-size', String(size))
  }, [])

  const tabsRef = useRef<MobileTab[]>(tabs)
  tabsRef.current = tabs

  const fetchSeqRef = useRef(0)

  useEffect(() => {
    checkAuth().then((ok) => setIsAuthenticated(ok))
  }, [])

  const fetchTree = useCallback(async () => {
    const seq = ++fetchSeqRef.current
    setLoading(true)
    try {
      const headers = getAuthHeaders()
      const res = await fetch('/api/tmux/tree', { headers })
      if (!res.ok) throw new Error('Failed to fetch tree')
      const data = await res.json()

      if (seq !== fetchSeqRef.current) return

      const newSessions: TmuxSession[] = data.data?.sessions || data.sessions || []
      setSessions(newSessions)
      setError(null)

      const allIds = getAllPaneIds(newSessions)
      const currentTabs = tabsRef.current
      const validTabs = currentTabs.filter((t) => allIds.has(t.paneId))
      if (validTabs.length !== currentTabs.length) {
        setTabs(validTabs)
        setActiveTabId((prev) => {
          if (prev && validTabs.some((t) => t.id === prev)) return prev
          return validTabs[0]?.id ?? null
        })
      }
    } catch (err) {
      if (seq !== fetchSeqRef.current) return
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      if (seq === fetchSeqRef.current) {
        setLoading(false)
      }
    }
  }, [])

  const fetchGroups = useCallback(async (profileKey: string) => {
    try {
      const headers = getAuthHeaders()
      const res = await fetch(`/api/groups?profile_key=${encodeURIComponent(profileKey)}`, {
        headers,
      })
      if (!res.ok) throw new Error('Failed to fetch groups')
      const data = await res.json()
      setGroups(data.data?.groups || data.groups || [])
    } catch (_err) {
      setGroups([])
    }
  }, [])

  useEffect(() => {
    if (isAuthenticated) {
      fetchTree()
    }
  }, [isAuthenticated, fetchTree])

  const handleProfileChange = useCallback(
    (profile: Profile) => {
      setCurrentProfile(profile)
      setGroups([])
      fetchTree()
      fetchGroups(profile.profile_key)
    },
    [fetchTree, fetchGroups],
  )

  const handleGroupsChanged = useCallback(() => {
    fetchTree()
    if (currentProfile) {
      fetchGroups(currentProfile.profile_key)
    }
  }, [fetchTree, fetchGroups, currentProfile])

  const handleSelectPane = useCallback(
    (paneId: string, paneName: string) => {
      setTabs((prev) => {
        const existing = prev.find((t) => t.paneId === paneId)
        if (existing) {
          setActiveTabId(existing.id)
          return prev
        }
        const sessionName = getSessionForPane(sessions, paneId) || ''
        const newTab: MobileTab = {
          id: `tab-${paneId}`,
          paneId,
          title: paneName,
          session: sessionName,
        }
        setActiveTabId(newTab.id)
        return [...prev, newTab]
      })
      setDrawerOpen(false)
    },
    [sessions],
  )

  const handlePaneStatusClick = useCallback((paneKey: string) => {
    setTaskHistoryPaneKey(paneKey)
    setRightPanelOpen(true)
  }, [])

  const handleCloseTab = useCallback((tabId: string) => {
    setTabs((prev) => {
      const idx = prev.findIndex((t) => t.id === tabId)
      const next = prev.filter((t) => t.id !== tabId)
      setActiveTabId((prevActive) => {
        if (prevActive !== tabId) return prevActive
        if (next.length === 0) return null
        const newIdx = Math.min(idx, next.length - 1)
        return next[newIdx]?.id ?? null
      })
      return next
    })
  }, [])

  const handleSelectTab = useCallback((tabId: string) => {
    setActiveTabId(tabId)
  }, [])

  const handleLogout = useCallback(async () => {
    await logout()
    setIsAuthenticated(false)
    setSessions([])
    setTabs([])
    setActiveTabId(null)
    setCurrentProfile(null)
    setGroups([])
  }, [])

  const toggleDrawer = useCallback(() => {
    setDrawerOpen((prev) => !prev)
  }, [])

  const activeTab = tabs.find((t) => t.id === activeTabId) ?? null
  const activePaneKey = useMemo(() => {
    return activeTab ? getPaneKey(sessions, activeTab.paneId) : null
  }, [activeTab, sessions])

  const toggleRightPanel = useCallback(() => {
    setRightPanelOpen((prev) => {
      const next = !prev
      if (next && !taskHistoryPaneKey) {
        setTaskHistoryPaneKey(activePaneKey)
      }
      return next
    })
  }, [taskHistoryPaneKey, activePaneKey])

  if (isAuthenticated === null) {
    return (
      <div className="flex h-full items-center justify-center bg-base-300 font-sans text-base-content/75">
        Loading…
      </div>
    )
  }

  if (!isAuthenticated) {
    return <LoginModal onLogin={() => setIsAuthenticated(true)} />
  }

  if (loading && sessions.length === 0) {
    return (
      <div className="flex h-full items-center justify-center bg-base-300 font-sans text-base-content/75">
        Loading sessions…
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center bg-base-300 font-sans text-error">
        {error}
      </div>
    )
  }

  const historyPaneKey = taskHistoryPaneKey ?? activePaneKey

  return (
    <div className="flex h-full w-screen flex-col overflow-hidden bg-base-300 text-base-content/75 text-[clamp(0.875rem,2.5vw,1rem)]">
      <a href="#mobile-content" className="skip-link">
        Skip to content
      </a>
      <header className="mobile-header flex h-12 shrink-0 items-center border-b border-base-200 bg-base-200 pl-3">
        <button
          className="btn btn-ghost btn-sm btn-circle mobile-menu-btn min-h-[44px] min-w-[44px] shrink-0 text-base-content/75"
          onClick={toggleDrawer}
          type="button"
          aria-label="Open menu"
        >
          <Menu size={24} />
        </button>
        {tabs.length > 0 ? (
          <div className="mobile-tabs-bar flex h-full flex-1 min-w-0 items-stretch overflow-x-auto touch-scroll scrollbar-hidden bg-transparent">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                className={`mobile-tab btn-reset flex h-full shrink-0 items-center gap-1.5 whitespace-nowrap border-r border-base-200 bg-transparent px-2.5 tap-none font-sans ${tab.id === activeTabId ? 'active bg-primary/10 border-b-2 border-b-primary' : ''}`}
                onClick={() => handleSelectTab(tab.id)}
              >
                <span className="mobile-tab-title max-w-[120px] truncate text-xs text-base-content/45">
                  {tab.title}
                </span>
                <button
                  className="btn btn-ghost btn-xs mobile-tab-close min-h-[44px] min-w-[44px] shrink-0 p-0 text-base-content/45"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleCloseTab(tab.id)
                  }}
                  type="button"
                  aria-label="Close tab"
                >
                  <X size={12} />
                </button>
              </button>
            ))}
          </div>
        ) : (
          <span className="mobile-title flex-1 truncate pl-3 text-sm font-medium font-sans">
            Select a pane
          </span>
        )}
        {activeTab && (
          <>
            <button
              className="btn btn-ghost btn-sm btn-circle mobile-menu-btn min-h-[44px] min-w-[44px] shrink-0 text-base-content/75"
              onClick={() => setImperialOpen(true)}
              type="button"
              title="Imperial Study"
              aria-label="Imperial Study"
            >
              <ScrollText size={22} />
            </button>
            <button
              className="btn btn-ghost btn-sm btn-circle mobile-menu-btn min-h-[44px] min-w-[44px] shrink-0 text-base-content/75"
              onClick={toggleRightPanel}
              type="button"
              title="Task history"
              aria-label="Task history"
            >
              <History size={22} />
            </button>
          </>
        )}
      </header>

      {(drawerOpen || rightPanelOpen || imperialOpen) && (
        <>
          {/* biome-ignore lint/a11y/noStaticElementInteractions: overlay click-outside-to-close pattern */}
          <div
            className="mobile-overlay fixed inset-0 z-[var(--z-overlay)] bg-black/50 tap-none"
            role="presentation"
            onClick={() => {
              setDrawerOpen(false)
              setRightPanelOpen(false)
              setImperialOpen(false)
            }}
          />
        </>
      )}

      {imperialOpen && (
        <div className="mobile-imperial-panel fixed inset-0 z-[200] flex flex-col overflow-hidden bg-base-100">
          <header className="mobile-imperial-header flex shrink-0 items-center justify-between border-b border-base-200 px-4 py-3 text-base font-semibold text-base-content">
            <span>Imperial Study</span>
            <button
              className="btn btn-ghost btn-sm btn-circle mobile-menu-btn min-h-[44px] min-w-[44px] shrink-0 text-base-content/75"
              onClick={() => setImperialOpen(false)}
              type="button"
            >
              <X size={22} />
            </button>
          </header>
          <ImperialStudyPanel activePaneKey={activePaneKey} />
        </div>
      )}

      <MobileDrawer
        open={drawerOpen}
        sessions={sessions}
        currentProfile={currentProfile}
        groups={groups}
        statusRefreshToken={statusRefreshToken}
        onProfileChange={handleProfileChange}
        onGroupsChanged={handleGroupsChanged}
        onSelectPane={handleSelectPane}
        onPaneStatusClick={handlePaneStatusClick}
        onClose={() => setDrawerOpen(false)}
        onRefresh={fetchTree}
        onLogout={handleLogout}
      />

      <aside
        className={`mobile-right-panel fixed right-0 top-0 bottom-0 z-[var(--z-drawer)] flex w-[85vw] max-w-[360px] flex-col overflow-hidden bg-base-200 transition-transform duration-[250ms] ease-out ${rightPanelOpen ? 'open translate-x-0' : 'translate-x-full'}`}
      >
        {rightPanelOpen && (
          <TaskHistoryPanel
            paneKey={historyPaneKey}
            onClose={() => setRightPanelOpen(false)}
            onStatusChange={() => setStatusRefreshToken((prev) => prev + 1)}
          />
        )}
      </aside>

      <main
        className="mobile-main flex min-h-0 flex-1 flex-col overflow-hidden pb-[env(safe-area-inset-bottom)]"
        id="mobile-content"
      >
        {tabs.length > 0 ? (
          <div className="mobile-tabs-content relative flex-1 min-h-0 overflow-hidden">
            {tabs.map((tab) => {
              const isActive = tab.id === activeTabId
              const isLive = liveTabIds.has(tab.id)

              return (
                <div
                  key={tab.id}
                  className={`mobile-tab-panel absolute inset-0 ${isActive ? 'visible' : 'hidden'}`}
                >
                  {isLive ? (
                    <MobileTerminal
                      session={tab.session}
                      pane={tab.paneId}
                      fontSize={fontSize}
                      onFontSizeChange={handleFontSizeChange}
                      voiceRef={voiceRef}
                      taskHistoryPaneKey={historyPaneKey}
                      onStatusChange={() => setStatusRefreshToken((prev) => prev + 1)}
                    />
                  ) : (
                    <div className="flex items-center justify-center h-full text-base-content/50 text-sm select-none">
                      Terminal suspended — tap to resume
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          <div className="mobile-placeholder flex flex-1 items-center justify-center text-sm font-sans text-base-content/45">
            <p className="flex items-center gap-2">
              Tap <Menu size={20} /> to select a terminal
            </p>
          </div>
        )}
      </main>
    </div>
  )
}
