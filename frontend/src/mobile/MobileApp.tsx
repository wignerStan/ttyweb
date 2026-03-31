import { History, Menu, ScrollText, X } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import useShakeDetect from '../hooks/useShakeDetect'
import useVisualViewport from '../hooks/useVisualViewport'
import { ImperialStudyPanel } from '../shared/components/imperial-study/components/ImperialStudyPanel'
import { LoginModal } from '../shared/components/LoginModal'
import { TaskHistoryPanel } from '../shared/components/TaskHistoryPanel'
import type { VoiceInputHandle } from '../shared/components/VoiceInput'
import { BUTTON_RESET } from '../shared/styles'
import type { OpenTab, Profile, SessionGroup, TmuxSession } from '../types'
import { checkAuth, getAuthHeader, logout } from '../utils/auth'
import { MobileDrawer } from './MobileDrawer'
import { MobileTerminal } from './MobileTerminal'
import './mobile.css'

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
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers.Authorization = auth
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
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers.Authorization = auth
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
    return <div className="mobile-loading">Loading...</div>
  }

  if (!isAuthenticated) {
    return <LoginModal onLogin={() => setIsAuthenticated(true)} />
  }

  if (loading && sessions.length === 0) {
    return <div className="mobile-loading">Loading sessions...</div>
  }

  if (error) {
    return <div className="mobile-error">{error}</div>
  }

  const historyPaneKey = taskHistoryPaneKey ?? activePaneKey

  return (
    <div className="mobile-app">
      <a href="#mobile-content" className="skip-link">
        Skip to content
      </a>
      <header className="mobile-header">
        <button
          className="mobile-menu-btn"
          onClick={toggleDrawer}
          type="button"
          aria-label="Open menu"
        >
          <Menu size={24} />
        </button>
        {tabs.length > 0 ? (
          <div className="mobile-tabs-bar">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                className={`mobile-tab ${tab.id === activeTabId ? 'active' : ''}`}
                style={BUTTON_RESET}
                onClick={() => handleSelectTab(tab.id)}
              >
                <span className="mobile-tab-title">{tab.title}</span>
                <button
                  className="mobile-tab-close"
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
          <span className="mobile-title">Select a pane</span>
        )}
        {activeTab && (
          <>
            <button
              className="mobile-menu-btn"
              onClick={() => setImperialOpen(true)}
              type="button"
              title="Imperial Study"
              aria-label="Imperial Study"
            >
              <ScrollText size={22} />
            </button>
            <button
              className="mobile-menu-btn"
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
            className="mobile-overlay"
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
        <div className="mobile-imperial-panel">
          <header className="mobile-imperial-header">
            <span>Imperial Study</span>
            <button
              className="mobile-menu-btn"
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

      <aside className={`mobile-right-panel ${rightPanelOpen ? 'open' : ''}`}>
        {rightPanelOpen && (
          <TaskHistoryPanel
            paneKey={historyPaneKey}
            onClose={() => setRightPanelOpen(false)}
            onStatusChange={() => setStatusRefreshToken((prev) => prev + 1)}
          />
        )}
      </aside>

      <main className="mobile-main" id="mobile-content">
        {tabs.length > 0 ? (
          <div className="mobile-tabs-content">
            {activeTab && (
              <div key={activeTab.id} className="mobile-tab-panel visible">
                <MobileTerminal
                  session={activeTab.session}
                  pane={activeTab.paneId}
                  fontSize={fontSize}
                  onFontSizeChange={handleFontSizeChange}
                  voiceRef={voiceRef}
                  taskHistoryPaneKey={historyPaneKey}
                  onStatusChange={() => setStatusRefreshToken((prev) => prev + 1)}
                />
              </div>
            )}
          </div>
        ) : (
          <div className="mobile-placeholder">
            <p>
              Tap <Menu size={20} /> to select a terminal
            </p>
          </div>
        )}
      </main>
    </div>
  )
}
