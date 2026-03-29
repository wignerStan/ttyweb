import React, { useState, useEffect, useCallback } from 'react';
import { Routes, Route } from 'react-router-dom';
import { Sidebar } from './components/Sidebar';
import { TerminalTab } from './components/TerminalTab';
import { KanbanBoard } from './kanban';
import { FloatingImperialStudy } from './shared/components/imperial-study/components/FloatingImperialStudy';
import MobileApp from './mobile/MobileApp';

interface Tab {
  id: string;
  session: string;
  pane: string;
}

type ViewMode = 'terminal' | 'kanban';

function DesktopLayout() {
  const [tabs, setTabs] = useState<Tab[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>('terminal');
  const [imperialStudyOpen, setImperialStudyOpen] = useState(true);

  const openTab = useCallback((session: string, pane?: string) => {
    const id = pane ? `${session}:${pane}` : session;
    setTabs((prev) => {
      if (prev.some((t) => t.id === id)) return prev;
      return [...prev, { id, session, pane: pane ?? '' }];
    });
    setActiveTabId(id);
  }, []);

  const closeTab = useCallback((tabId: string) => {
    setTabs((prev) => {
      const next = prev.filter((t) => t.id !== tabId);
      if (activeTabId === tabId && next.length > 0) {
        setActiveTabId(next[next.length - 1]?.id ?? null);
      }
      return next;
    });
  }, [activeTabId]);

  // Open default tab on mount
  useEffect(() => {
    openTab('ttyweb');
  }, [openTab]);

  const activeTab = tabs.find((t) => t.id === activeTabId) ?? null;

  return (
    <div style={styles.container}>
      {imperialStudyOpen && (
        <FloatingImperialStudy
          activePaneKey={activeTab?.id ?? null}
          onClose={() => setImperialStudyOpen(false)}
        />
      )}
      {sidebarOpen && (
        <div style={styles.sidebar}>
          <Sidebar onSelect={openTab} />
        </div>
      )}
      <div style={styles.main}>
        <div style={styles.tabBar}>
          <button onClick={() => setSidebarOpen(!sidebarOpen)} style={styles.toggleBtn}>
            {sidebarOpen ? '\u25C0' : '\u25B6'}
          </button>
          {tabs.map((tab) => (
            <div
              key={tab.id}
              style={{
                ...styles.tab,
                ...(tab.id === activeTabId ? styles.tabActive : {}),
              }}
              onClick={() => {
                setViewMode('terminal');
                setActiveTabId(tab.id);
              }}
            >
              <span style={styles.tabLabel}>{tab.id}</span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  closeTab(tab.id);
                }}
                style={styles.tabClose}
              >
                \u00D7
              </button>
            </div>
          ))}
          <div
            style={{
              ...styles.tab,
              ...(viewMode === 'kanban' ? styles.tabActive : {}),
            }}
            onClick={() => setViewMode('kanban')}
          >
            <span style={styles.tabLabel}>Kanban</span>
          </div>
        </div>
        <div style={styles.terminalArea}>
          {viewMode === 'kanban' ? (
            <KanbanBoard />
          ) : (
            activeTab && (
              <TerminalTab
                key={activeTab.id}
                session={activeTab.session}
                pane={activeTab.pane}
              />
            )
          )}
        </div>
      </div>
    </div>
  );
}

export function App() {
  return (
    <Routes>
      <Route path="/m" element={<MobileApp />} />
      <Route path="/*" element={<DesktopLayout />} />
    </Routes>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    height: '100vh',
    width: '100vw',
    backgroundColor: '#1a1b26',
    color: '#a9b1d6',
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
    overflow: 'hidden',
  },
  sidebar: {
    width: '260px',
    minWidth: '260px',
    borderRight: '1px solid #24283b',
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
    backgroundColor: '#16161e',
    borderBottom: '1px solid #24283b',
    height: '36px',
    paddingLeft: '4px',
    overflow: 'auto',
    flexShrink: 0,
  },
  toggleBtn: {
    background: 'none',
    border: 'none',
    color: '#a9b1d6',
    cursor: 'pointer',
    fontSize: '12px',
    padding: '4px 8px',
    marginRight: '4px',
  },
  tab: {
    display: 'flex',
    alignItems: 'center',
    padding: '4px 12px',
    cursor: 'pointer',
    borderRight: '1px solid #24283b',
    fontSize: '12px',
    whiteSpace: 'nowrap' as const,
  },
  tabActive: {
    backgroundColor: '#1a1b26',
    color: '#c0caf5',
  },
  tabLabel: {
    marginRight: '6px',
  },
  tabClose: {
    background: 'none',
    border: 'none',
    color: '#565f89',
    cursor: 'pointer',
    fontSize: '14px',
    lineHeight: '1',
    padding: '0',
  },
  terminalArea: {
    flex: 1,
    overflow: 'hidden',
  },
};
