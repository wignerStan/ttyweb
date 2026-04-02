import type React from 'react'
import { lazy, Suspense, useCallback, useEffect, useState } from 'react'
import { Route, Routes } from 'react-router-dom'
import { Sidebar } from './components/Sidebar'
import { TerminalTab } from './components/TerminalTab'
import type { AISession } from './conversations/types'
import MobileApp from './mobile/MobileApp'
import { LazyFallback } from './shared/components/LazyFallback'
import { NotificationProvider } from './shared/components/NotificationProvider'
import { ThemeToggle } from './shared/components/ThemeToggle'

const ConversationList = lazy(() =>
  import('./conversations/ConversationList').then((m) => ({ default: m.ConversationList })),
)
const ConversationViewer = lazy(() =>
  import('./conversations/ConversationViewer').then((m) => ({ default: m.ConversationViewer })),
)
const FloatingImperialStudy = lazy(() =>
  import('./shared/components/imperial-study/components/FloatingImperialStudy').then((m) => ({
    default: m.FloatingImperialStudy,
  })),
)
const NotepadPanel = lazy(() =>
  import('./notepad/NotepadPanel').then((m) => ({ default: m.NotepadPanel })),
)

const KanbanBoard = lazy(() => import('./kanban').then((m) => ({ default: m.KanbanBoard })))

type AppView = 'terminal' | 'conversations'

const NOTEPAD_TAB_ID = '__notepad__'

interface Tab {
  id: string
  session: string
  pane: string
  type?: 'terminal' | 'notepad'
}

type ViewMode = 'terminal' | 'kanban'

function DesktopLayout() {
  const [tabs, setTabs] = useState<Tab[]>([])
  const [activeTabId, setActiveTabId] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [viewMode, setViewMode] = useState<ViewMode>('terminal')
  const [imperialStudyOpen, setImperialStudyOpen] = useState(true)
  const [activeView, setActiveView] = useState<AppView>('terminal')
  const [selectedSession, setSelectedSession] = useState<AISession | null>(null)

  const openNotepad = useCallback(() => {
    setTabs((prev) => {
      if (prev.some((t) => t.id === NOTEPAD_TAB_ID)) return prev
      return [...prev, { id: NOTEPAD_TAB_ID, type: 'notepad' as const, session: '', pane: '' }]
    })
    setViewMode('terminal')
    setActiveTabId(NOTEPAD_TAB_ID)
  }, [])

  const openTab = useCallback((session: string, pane?: string) => {
    const id = pane ? `${session}:${pane}` : session
    setTabs((prev) => {
      if (prev.some((t) => t.id === id)) return prev
      return [...prev, { id, session, pane: pane ?? '', type: 'terminal' as const }]
    })
    setViewMode('terminal')
    setActiveTabId(id)
  }, [])

  const closeTab = useCallback((tabId: string) => {
    setTabs((prev) => prev.filter((t) => t.id !== tabId))
  }, [])

  // Auto-select last tab when active tab is removed
  useEffect(() => {
    if (activeTabId && !tabs.some((t) => t.id === activeTabId)) {
      setActiveTabId(tabs.length > 0 ? (tabs[tabs.length - 1]?.id ?? null) : null)
    }
  }, [tabs, activeTabId])

  // Open default tab on mount
  useEffect(() => {
    openTab('ttyweb')
  }, [openTab])

  const activeTab = tabs.find((t) => t.id === activeTabId) ?? null

  return (
    <div style={styles.container}>
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      {imperialStudyOpen && (
        <Suspense fallback={null}>
          <FloatingImperialStudy
            activePaneKey={activeTab?.id ?? null}
            onClose={() => setImperialStudyOpen(false)}
          />
        </Suspense>
      )}
      {sidebarOpen && (
        <nav style={styles.sidebar} aria-label="Session navigation">
          <Sidebar onSelect={openTab} />
        </nav>
      )}
      <div style={styles.main}>
        <div style={styles.tabBar}>
          <button
            type="button"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            style={styles.toggleBtn}
            aria-label="Toggle sidebar"
          >
            {sidebarOpen ? '\u25C0' : '\u25B6'}
          </button>
          <button
            type="button"
            className="btn-reset"
            style={{
              ...styles.viewTab,
              ...(activeView === 'terminal' ? styles.viewTabActive : {}),
            }}
            onClick={() => setActiveView('terminal')}
          >
            Terminal
          </button>
          <button
            type="button"
            className="btn-reset"
            style={{
              ...styles.viewTab,
              ...(activeView === 'conversations' ? styles.viewTabActive : {}),
            }}
            onClick={() => setActiveView('conversations')}
          >
            Conversations
          </button>
          {activeView === 'terminal' &&
            tabs.map((tab) => (
              <button
                key={tab.id}
                type="button"
                className="btn-reset"
                style={{
                  ...styles.tab,
                  ...(tab.id === activeTabId ? styles.tabActive : {}),
                }}
                onClick={() => {
                  setViewMode('terminal')
                  setActiveTabId(tab.id)
                }}
              >
                <span style={styles.tabLabel}>{tab.type === 'notepad' ? 'Notepad' : tab.id}</span>
                {tab.id !== NOTEPAD_TAB_ID && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      closeTab(tab.id)
                    }}
                    style={styles.tabClose}
                    title="Close tab"
                    aria-label="Close tab"
                  >
                    \u00D7
                  </button>
                )}
              </button>
            ))}
          <button
            type="button"
            className="btn-reset"
            style={{
              ...styles.tab,
              ...(viewMode === 'kanban' ? styles.tabActive : {}),
            }}
            onClick={() => setViewMode('kanban')}
          >
            <span style={styles.tabLabel}>Kanban</span>
          </button>
          <button
            type="button"
            onClick={openNotepad}
            style={styles.notepadBtn}
            title="Open Notepad"
            aria-label="Open notepad"
          >
            <span style={{ fontSize: '14px' }}>{'\u270E'}</span>
          </button>
          <ThemeToggle />
        </div>
        <div style={styles.contentArea} id="main-content">
          {activeView === 'terminal' && (
            <div style={styles.terminalArea}>
              {viewMode === 'kanban' ? (
                <Suspense fallback={<LazyFallback />}>
                  <KanbanBoard />
                </Suspense>
              ) : (
                <>
                  {activeTab && activeTab.type === 'notepad' && (
                    <Suspense fallback={<LazyFallback />}>
                      <NotepadPanel key={NOTEPAD_TAB_ID} />
                    </Suspense>
                  )}
                  {activeTab && activeTab.type === 'terminal' && (
                    <TerminalTab
                      key={activeTab.id}
                      session={activeTab.session}
                      pane={activeTab.pane}
                    />
                  )}
                </>
              )}
            </div>
          )}
          {activeView === 'conversations' && (
            <div style={styles.conversationsLayout}>
              <div style={styles.conversationsListPane}>
                <Suspense fallback={<LazyFallback />}>
                  <ConversationList
                    selectedSessionId={selectedSession?.id ?? null}
                    onSelectSession={setSelectedSession}
                  />
                </Suspense>
              </div>
              <div style={styles.conversationsViewerPane}>
                <Suspense fallback={<LazyFallback />}>
                  <ConversationViewer
                    sessionId={selectedSession?.id ?? null}
                    sessionInfo={selectedSession}
                    onBack={() => setSelectedSession(null)}
                  />
                </Suspense>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export function App() {
  return (
    <NotificationProvider>
      <Routes>
        <Route path="/m" element={<MobileApp />} />
        <Route path="/*" element={<DesktopLayout />} />
      </Routes>
    </NotificationProvider>
  )
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    height: '100vh',
    width: '100vw',
    backgroundColor: 'var(--bg-primary)',
    color: 'var(--text-secondary)',
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
    overflow: 'hidden',
  },
  sidebar: {
    width: '260px',
    minWidth: '260px',
    borderRight: '1px solid var(--border-color)',
    overflow: 'auto',
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
  },
  tabBar: {
    display: 'flex',
    alignItems: 'center',
    backgroundColor: 'var(--bg-secondary)',
    borderBottom: '1px solid var(--border-color)',
    height: '36px',
    paddingLeft: '4px',
    overflow: 'auto',
    flexShrink: 0,
  },
  toggleBtn: {
    background: 'none',
    border: 'none',
    color: 'var(--text-secondary)',
    cursor: 'pointer',
    fontSize: '12px',
    padding: '4px 8px',
    marginRight: '4px',
  },
  viewTab: {
    display: 'flex',
    alignItems: 'center',
    padding: '4px 14px',
    cursor: 'pointer',
    fontSize: '12px',
    fontWeight: 600,
    whiteSpace: 'nowrap' as const,
    color: 'var(--text-muted)',
    borderRight: '1px solid var(--border-color)',
  },
  viewTabActive: {
    backgroundColor: 'var(--bg-primary)',
    color: 'var(--text-primary)',
  },
  tab: {
    display: 'flex',
    alignItems: 'center',
    padding: '4px 12px',
    cursor: 'pointer',
    borderRight: '1px solid var(--border-color)',
    fontSize: '12px',
    whiteSpace: 'nowrap' as const,
  },
  tabActive: {
    backgroundColor: 'var(--bg-primary)',
    color: 'var(--text-primary)',
  },
  tabLabel: {
    marginRight: '6px',
  },
  tabClose: {
    background: 'none',
    border: 'none',
    color: 'var(--text-muted)',
    cursor: 'pointer',
    fontSize: '14px',
    lineHeight: '1',
    padding: '0',
  },
  notepadBtn: {
    background: 'none',
    border: '1px solid var(--border-light)',
    color: 'var(--text-muted)',
    cursor: 'pointer',
    fontSize: '12px',
    padding: '2px 8px',
    marginLeft: '4px',
    borderRadius: '2px',
    display: 'flex',
    alignItems: 'center',
  },
  contentArea: {
    flex: 1,
    overflow: 'hidden',
  },
  terminalArea: {
    flex: 1,
    overflow: 'hidden',
  },
  conversationsLayout: {
    display: 'flex',
    height: '100%',
    overflow: 'hidden',
  },
  conversationsListPane: {
    width: '320px',
    minWidth: '320px',
    borderRight: '1px solid var(--border-color)',
    overflow: 'hidden',
  },
  conversationsViewerPane: {
    flex: 1,
    overflow: 'hidden',
  },
}
