import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { type FSEntry, useFileBrowser } from './useFileBrowser'

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    fontSize: '13px',
    color: '#a9b1d6',
    backgroundColor: '#1a1b26',
    overflow: 'hidden',
  },
  breadcrumb: {
    display: 'flex',
    alignItems: 'center',
    padding: '8px 12px',
    backgroundColor: '#16161e',
    borderBottom: '1px solid #24283b',
    flexShrink: 0,
    overflow: 'auto',
    whiteSpace: 'nowrap' as const,
  },
  breadcrumbSegment: {
    background: 'none',
    border: 'none',
    padding: '2px 4px',
    color: '#7aa2f7',
    cursor: 'pointer',
    fontSize: '13px',
    fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
  },
  breadcrumbSeparator: {
    color: '#565f89',
    margin: '0 2px',
    userSelect: 'none' as const,
  },
  listContainer: {
    flex: 1,
    overflow: 'auto',
  },
  row: {
    display: 'flex',
    alignItems: 'center',
    padding: '4px 12px',
    cursor: 'pointer',
    borderBottom: '1px solid #1f2335',
  },
  rowHover: {
    backgroundColor: '#1f2335',
  },
  nameCell: {
    flex: 1,
    minWidth: 0,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap' as const,
  },
  sizeCell: {
    width: '100px',
    textAlign: 'right' as const,
    color: '#565f89',
    flexShrink: 0,
  },
  timeCell: {
    width: '160px',
    textAlign: 'right' as const,
    color: '#565f89',
    flexShrink: 0,
    paddingLeft: '16px',
  },
  dirIcon: {
    color: '#7aa2f7',
    marginRight: '6px',
  },
  fileIcon: {
    color: '#565f89',
    marginRight: '6px',
  },
  loading: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    color: '#565f89',
  },
  error: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    color: '#f7768e',
    padding: '16px',
  },
  empty: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    color: '#565f89',
  },
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

interface FileBrowserProps {
  initialPath?: string
}

export function FileBrowser({ initialPath = '.' }: FileBrowserProps) {
  const { entries, loading, error, navigateTo } = useFileBrowser()
  const [currentPath, setCurrentPath] = useState(initialPath)
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null)
  const initialPathRef = useRef(initialPath)

  // Load initial path on mount.
  useEffect(() => {
    void navigateTo(initialPathRef.current)
  }, [navigateTo])

  const handleEntryClick = useCallback(
    (entry: FSEntry) => {
      if (!entry.is_dir) return
      const next = currentPath === '.' ? entry.name : `${currentPath}/${entry.name}`
      setCurrentPath(next)
      void navigateTo(next)
    },
    [currentPath, navigateTo],
  )

  const handleBreadcrumbClick = useCallback(
    (segment: string) => {
      setCurrentPath(segment)
      void navigateTo(segment)
    },
    [navigateTo],
  )

  const breadcrumbSegments = useMemo(() => {
    if (currentPath === '.') return [{ label: '.', path: '.' }]
    const parts = currentPath.split('/')
    return parts.map((part, i) => ({
      label: part,
      path: parts.slice(0, i + 1).join('/'),
    }))
  }, [currentPath])

  if (loading && entries.length === 0) {
    return (
      <div style={styles.container}>
        <div style={styles.loading}>Loading...</div>
      </div>
    )
  }

  if (error && entries.length === 0) {
    return (
      <div style={styles.container}>
        <div style={styles.breadcrumb}>
          {breadcrumbSegments.map((seg, i) => (
            <span key={seg.path}>
              {i > 0 && <span style={styles.breadcrumbSeparator}>/</span>}
              <button
                type="button"
                style={styles.breadcrumbSegment}
                onClick={() => handleBreadcrumbClick(seg.path)}
              >
                {seg.label}
              </button>
            </span>
          ))}
        </div>
        <div style={styles.error}>{error}</div>
      </div>
    )
  }

  return (
    <div style={styles.container}>
      <div style={styles.breadcrumb}>
        {breadcrumbSegments.map((seg, i) => (
          <span key={seg.path}>
            {i > 0 && <span style={styles.breadcrumbSeparator}>/</span>}
            <button
              type="button"
              style={styles.breadcrumbSegment}
              onClick={() => handleBreadcrumbClick(seg.path)}
            >
              {seg.label}
            </button>
          </span>
        ))}
      </div>
      <div style={styles.listContainer}>
        {entries.length === 0 && !loading && <div style={styles.empty}>Empty directory</div>}
        {entries.map((entry, index) => (
          <button
            key={entry.name}
            type="button"
            className="btn-reset"
            style={{
              ...styles.row,
              ...(hoveredIndex === index ? styles.rowHover : {}),
              cursor: entry.is_dir ? 'pointer' : 'default',
            }}
            onClick={() => handleEntryClick(entry)}
            onMouseEnter={() => setHoveredIndex(index)}
            onMouseLeave={() => setHoveredIndex(null)}
          >
            <span style={styles.nameCell}>
              <span style={entry.is_dir ? styles.dirIcon : styles.fileIcon}>
                {entry.is_dir ? '\u{1F4C1}' : '\u{1F4C4}'}
              </span>
              {entry.name}
            </span>
            <span style={styles.timeCell}>{entry.mod_time}</span>
            <span style={styles.sizeCell}>{entry.is_dir ? '-' : formatSize(entry.size)}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
