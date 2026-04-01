import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  FolderInput,
  FolderMinus,
  FolderPlus,
  GripVertical,
  Loader2,
  MoreHorizontal,
  Pencil,
  RotateCcw,
  Terminal,
  XCircle,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useLongPress } from '../../../hooks/useLongPress'
import type { SessionGroup, TmuxSession } from '../../../types'
import { ConfirmDialog } from '../ConfirmDialog'
import { useNotification } from '../NotificationProvider'
import { StatusBadge } from '../StatusBadge'
import { assignSessionGroup, createGroup, rebuildSession, renameWindow } from './api'
import { useTmuxTree } from './TmuxTreeContext'

// ── TreeItem (shared with TmuxTree) ───────────────────────────────────────

export interface TreeItem {
  id: string
  type: 'session' | 'group'
  session?: TmuxSession
  group?: SessionGroup
  groupId: number | null
  sortOrder: number
}

// ── buildPaneKey ──────────────────────────────────────────────────────────

function buildPaneKey(sessionName: string, windowIndex: number, paneId: string): string {
  return `${sessionName}:${windowIndex}:${paneId}`
}

// ── DragHandle ────────────────────────────────────────────────────────────

function DragHandle() {
  return (
    <span className="drag-handle" role="img" aria-label="Drag to reorder">
      <GripVertical size={14} />
    </span>
  )
}

// ── QuickGroupMenu ────────────────────────────────────────────────────────

interface QuickGroupMenuProps {
  sessionName: string
  currentGroupId: number | null
  groups: SessionGroup[]
  profileKey: string
  position: { x: number; y: number }
  onClose: () => void
  onDone: () => void
}

function QuickGroupMenu({
  sessionName,
  currentGroupId,
  groups,
  profileKey,
  position,
  onClose,
  onDone,
}: QuickGroupMenuProps) {
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (creating && inputRef.current) inputRef.current.focus()
  }, [creating])

  // Adjust menu position to stay in viewport
  useEffect(() => {
    if (!menuRef.current) return
    const rect = menuRef.current.getBoundingClientRect()
    if (rect.right > window.innerWidth) {
      menuRef.current.style.left = `${window.innerWidth - rect.width - 8}px`
    }
    if (rect.bottom > window.innerHeight) {
      menuRef.current.style.top = `${window.innerHeight - rect.height - 8}px`
    }
  }, [])

  const assignToGroup = async (groupId: number | null) => {
    if (loading) return
    setLoading(true)
    try {
      await assignSessionGroup(sessionName, profileKey, groupId)
      onDone()
    } catch (_err) {
    } finally {
      setLoading(false)
      onClose()
    }
  }

  const createAndAssign = async () => {
    if (!newName.trim() || loading) return
    setLoading(true)
    try {
      const data = await createGroup(profileKey, newName.trim())
      if (data.id) {
        await assignSessionGroup(sessionName, profileKey, data.id)
        onDone()
      }
    } catch (_err) {
    } finally {
      setLoading(false)
      onClose()
    }
  }

  return (
    <>
      {/* biome-ignore lint/a11y/noStaticElementInteractions: context menu backdrop */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: context menu backdrop */}
      <div
        className="quick-group-backdrop"
        onClick={onClose}
        onTouchEnd={(e) => {
          e.preventDefault()
          onClose()
        }}
      />
      <div ref={menuRef} className="quick-group-menu" style={{ left: position.x, top: position.y }}>
        <div className="quick-group-title">移动到分组</div>

        {groups.length > 0 &&
          groups.map((g) => (
            <button
              type="button"
              key={g.id}
              className={`quick-group-item ${g.id === currentGroupId ? 'current' : ''}`}
              onClick={() => assignToGroup(g.id)}
              disabled={loading || g.id === currentGroupId}
            >
              <FolderInput size={14} />
              <span>{g.group_name}</span>
              {g.id === currentGroupId && <span className="quick-group-check">✓</span>}
            </button>
          ))}

        {currentGroupId !== null && (
          <button
            type="button"
            className="quick-group-item ungroup"
            onClick={() => assignToGroup(null)}
            disabled={loading}
          >
            <FolderMinus size={14} />
            <span>移出分组</span>
          </button>
        )}

        <div className="quick-group-divider" />

        {creating ? (
          <div className="quick-group-create-row">
            <input
              ref={inputRef}
              type="text"
              className="quick-group-input"
              placeholder="分组名称..."
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') createAndAssign()
                if (e.key === 'Escape') {
                  setCreating(false)
                  setNewName('')
                }
              }}
              disabled={loading}
            />
            <button
              type="button"
              className="quick-group-confirm"
              onClick={createAndAssign}
              disabled={loading || !newName.trim()}
            >
              ✓
            </button>
          </div>
        ) : (
          <button
            type="button"
            className="quick-group-item create"
            onClick={() => setCreating(true)}
            disabled={loading}
          >
            <FolderPlus size={14} />
            <span>新建分组</span>
          </button>
        )}
      </div>
    </>
  )
}

// ── SortableSession ───────────────────────────────────────────────────────

interface SortableSessionProps {
  item: TreeItem
  session: TmuxSession
  isInGroup: boolean
  isOver?: boolean
}

export function SortableSession({ item, session, isInGroup, isOver }: SortableSessionProps) {
  const { statusMap, actions, meta } = useTmuxTree()
  const { onSelectPane, onPaneContextMenu, onPaneStatusClick, onRefresh, onGroupChanged } = actions
  const { profileKey, groups, defaultExpanded } = meta
  const { notify } = useNotification()
  const [expanded, setExpanded] = useState(defaultExpanded)
  const [editingWindowIndex, setEditingWindowIndex] = useState<number | null>(null)
  const [editWindowName, setEditWindowName] = useState('')
  const [quickGroupMenu, setQuickGroupMenu] = useState<{ x: number; y: number } | null>(null)
  const [rebuilding, setRebuilding] = useState(false)
  const [showRebuildConfirm, setShowRebuildConfirm] = useState(false)

  const handleRebuildClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation()
      if (rebuilding) return
      setShowRebuildConfirm(true)
    },
    [rebuilding],
  )

  const handleRebuildConfirm = useCallback(async () => {
    setShowRebuildConfirm(false)
    setRebuilding(true)
    try {
      const result = await rebuildSession(session.sessionName)
      if (result.ok) {
        onRefresh()
      } else {
        notify({
          type: 'error',
          title: 'Rebuild Failed',
          message: result.message || 'Unknown error',
          paneKey: '',
        })
        return
      }
    } catch (err) {
      notify({
        type: 'error',
        title: 'Rebuild Failed',
        message: err instanceof Error ? err.message : 'Network error',
        paneKey: '',
      })
      return
    } finally {
      setRebuilding(false)
    }
  }, [session.sessionName, onRefresh, notify])

  // Aggregate session-level status from all panes
  const sessionStatus = useMemo(() => {
    let inProgress = 0
    let done = 0
    let failed = 0
    let waiting = 0
    let total = 0
    session.windows.forEach((w) => {
      w.panes.forEach((p) => {
        const key = buildPaneKey(session.sessionName, w.windowIndex, p.paneId)
        const st = statusMap[key] || 'idle'
        total++
        if (st === 'in_progress') inProgress++
        else if (st === 'done') done++
        else if (st === 'failed') failed++
        else if (st === 'waiting') waiting++
      })
    })
    return { inProgress, done, failed, waiting, total }
  }, [session, statusMap])

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  const handleLongPress = useCallback(
    (pos: { x: number; y: number }) => {
      if (!profileKey || !groups) return
      setQuickGroupMenu(pos)
    },
    [profileKey, groups],
  )

  const longPress = useLongPress<{ x: number; y: number }>({
    threshold: 500,
    onLongPress: handleLongPress,
    transform: (e) => {
      if ('touches' in e) {
        const t = e.touches[0]
        if (!t) return { x: 0, y: 0 }
        return { x: t.clientX, y: t.clientY }
      }
      return { x: (e as React.MouseEvent).clientX, y: (e as React.MouseEvent).clientY }
    },
  })

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`session-node ${isDragging ? 'dragging' : ''} ${isOver ? 'drop-target' : ''} ${isInGroup ? 'in-group' : ''}`}
    >
      <div className="session-row" {...longPress}>
        <span {...attributes} {...listeners}>
          <DragHandle />
        </span>
        <button
          type="button"
          className="expand-btn"
          onClick={() => setExpanded(!expanded)}
          aria-label={expanded ? 'Collapse session' : 'Expand session'}
        >
          {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        </button>
        <Terminal size={14} style={{ color: 'var(--color-primary)' }} />
        <span className="session-name">{session.sessionName}</span>
        {(sessionStatus.inProgress > 0 ||
          sessionStatus.done > 0 ||
          sessionStatus.failed > 0 ||
          sessionStatus.waiting > 0) && (
          <span className="session-status-summary">
            {sessionStatus.inProgress > 0 && (
              <span
                className="session-stat session-stat--progress"
                title={`${sessionStatus.inProgress} 进行中`}
              >
                <Loader2 size={10} className="spinning" />
                {sessionStatus.inProgress}
              </span>
            )}
            {sessionStatus.done > 0 && (
              <span
                className="session-stat session-stat--done"
                title={`${sessionStatus.done} 已完成`}
              >
                <CheckCircle2 size={10} />
                {sessionStatus.done}
              </span>
            )}
            {sessionStatus.failed > 0 && (
              <span
                className="session-stat session-stat--failed"
                title={`${sessionStatus.failed} 失败`}
              >
                <XCircle size={10} />
                {sessionStatus.failed}
              </span>
            )}
            {sessionStatus.waiting > 0 && (
              <span
                className="session-stat session-stat--waiting"
                title={`${sessionStatus.waiting} 等待中`}
              >
                <Clock size={10} />
                {sessionStatus.waiting}
              </span>
            )}
          </span>
        )}
        <button
          type="button"
          className="rebuild-btn"
          onClick={handleRebuildClick}
          disabled={rebuilding}
          title="Rebuild session"
        >
          {rebuilding ? <Loader2 size={12} className="spinning" /> : <RotateCcw size={12} />}
        </button>
      </div>

      {quickGroupMenu && profileKey && (
        <QuickGroupMenu
          sessionName={session.sessionName}
          currentGroupId={item.groupId}
          groups={groups}
          profileKey={profileKey}
          position={quickGroupMenu}
          onClose={() => setQuickGroupMenu(null)}
          onDone={() => {
            setQuickGroupMenu(null)
            onGroupChanged?.()
          }}
        />
      )}

      {expanded &&
        session.windows.map((window) => (
          <div key={window.windowId} className="window-node">
            <div className="window-row">
              {editingWindowIndex === window.windowIndex ? (
                <input
                  type="text"
                  name="window-name"
                  autoComplete="off"
                  className="window-name-input"
                  value={editWindowName}
                  onChange={(e) => setEditWindowName(e.target.value)}
                  onKeyDown={async (e) => {
                    if (e.key === 'Enter' && editWindowName.trim()) {
                      const ok = await renameWindow(
                        session.sessionName,
                        window.windowIndex,
                        editWindowName.trim(),
                      )
                      if (ok) onRefresh()
                      setEditingWindowIndex(null)
                    }
                    if (e.key === 'Escape') setEditingWindowIndex(null)
                  }}
                  onBlur={() => setEditingWindowIndex(null)}
                />
              ) : (
                <>
                  <span className="window-name">
                    {window.windowIndex}: {window.windowName}
                  </span>
                  <button
                    type="button"
                    className="window-rename-btn"
                    onClick={() => {
                      setEditWindowName(window.windowName)
                      setEditingWindowIndex(window.windowIndex)
                    }}
                    title="Rename window"
                    aria-label="Rename window"
                  >
                    <Pencil size={12} />
                  </button>
                </>
              )}
            </div>
            {window.panes.map((pane) => {
              const paneKey = buildPaneKey(session.sessionName, window.windowIndex, pane.paneId)
              const paneStatus = statusMap[paneKey] || 'idle'
              return (
                <>
                  {/* biome-ignore lint/a11y/useSemanticElements: tree pane node requires div for context menu */}
                  <div
                    key={pane.paneId}
                    className="pane-node"
                    onClick={() =>
                      onSelectPane(pane.paneId, `${session.sessionName}:${window.windowIndex}`)
                    }
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        onSelectPane(pane.paneId, `${session.sessionName}:${window.windowIndex}`)
                      }
                    }}
                    onContextMenu={(e) => {
                      e.preventDefault()
                      onPaneContextMenu?.(paneKey)
                    }}
                    role="button"
                    tabIndex={0}
                  >
                    <span className="pane-id">{pane.paneId}</span>
                    {/* biome-ignore lint/a11y/noStaticElementInteractions: pane status click target */}
                    {/* biome-ignore lint/a11y/useKeyWithClickEvents: pane status click target */}
                    <span
                      className="pane-status-clickable"
                      onClick={(e) => {
                        e.stopPropagation()
                        onPaneStatusClick?.(paneKey)
                      }}
                      title="View task history"
                    >
                      <StatusBadge status={paneStatus} size="small" />
                    </span>
                    <span className="pane-cmd">{pane.paneCommand}</span>
                    {onPaneContextMenu && (
                      <button
                        type="button"
                        className="pane-details-btn"
                        onClick={(e) => {
                          e.stopPropagation()
                          onPaneContextMenu(paneKey)
                        }}
                        title="View details"
                        aria-label="Pane details"
                      >
                        <MoreHorizontal size={14} />
                      </button>
                    )}
                  </div>
                </>
              )
            })}
          </div>
        ))}

      <ConfirmDialog
        open={showRebuildConfirm}
        title="Rebuild Session"
        message={`Rebuild session "${session.sessionName}"? This will kill all processes and recreate windows with the same directories.`}
        confirmLabel="Rebuild"
        variant="destructive"
        onConfirm={handleRebuildConfirm}
        onCancel={() => setShowRebuildConfirm(false)}
      />
    </div>
  )
}
