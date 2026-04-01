import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  type DragOverEvent,
  DragOverlay,
  type DragStartEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  Folder,
  FolderOpen,
  GripVertical,
  Loader2,
  RefreshCw,
  Terminal,
  XCircle,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { PaneStatus, SessionGroup, TmuxSession } from '../../types'
import { NewTmuxButton } from './NewTmuxButton'
import {
  fetchPaneStatuses,
  fetchProfileOrder,
  fetchTaskPaneStatuses,
  type OrderData,
  type SessionOrder,
  saveOrder,
} from './tmux/api'
import { SortableSession, type TreeItem } from './tmux/SortableSession'
import { TmuxTreeContext } from './tmux/TmuxTreeContext'

// ── buildPaneKey (used in allPaneKeys computation) ────────────────────────

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

// ── Props ─────────────────────────────────────────────────────────────────

interface Props {
  sessions: TmuxSession[]
  groups?: SessionGroup[]
  profileId?: number
  profileKey?: string
  onSelectPane: (paneId: string, paneName: string) => void
  onRefresh: () => void
  onOrderChange?: () => void
  onPaneContextMenu?: (paneKey: string) => void
  onPaneStatusClick?: (paneKey: string) => void
  statusRefreshToken?: number
  defaultExpanded?: boolean
}

// ── SortableGroup ─────────────────────────────────────────────────────────

interface SortableGroupProps {
  item: TreeItem
  group: SessionGroup
  children: React.ReactNode
  isOver?: boolean
}

function SortableGroup({ item, group, children, isOver }: SortableGroupProps) {
  const [expanded, setExpanded] = useState(true)

  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`group-node ${isDragging ? 'dragging' : ''} ${isOver ? 'drop-target-group' : ''}`}
    >
      <div className="group-row">
        <span {...attributes} {...listeners}>
          <DragHandle />
        </span>
        <button
          type="button"
          className="expand-btn"
          onClick={() => setExpanded(!expanded)}
          aria-label={expanded ? 'Collapse group' : 'Expand group'}
        >
          {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        </button>
        {expanded ? (
          <FolderOpen size={14} style={{ color: 'var(--color-primary)' }} />
        ) : (
          <Folder size={14} style={{ color: 'var(--color-primary)' }} />
        )}
        <span className="group-name">{group.group_name}</span>
        <span className="group-count">{group.session_count}</span>
      </div>

      {expanded && <div className="group-children">{children}</div>}
    </div>
  )
}

// ── DragPreview ───────────────────────────────────────────────────────────

function DragPreview({ item }: { item: TreeItem | null }) {
  if (!item) return null

  if (item.type === 'session' && item.session) {
    return (
      <div className="drag-preview session-preview">
        <DragHandle />
        <ChevronRight size={12} />
        <Terminal size={14} style={{ color: 'var(--color-primary)' }} />
        <span className="session-name">{item.session.sessionName}</span>
      </div>
    )
  }

  if (item.type === 'group' && item.group) {
    return (
      <div className="drag-preview group-preview">
        <DragHandle />
        <ChevronDown size={12} />
        <FolderOpen size={14} style={{ color: 'var(--color-primary)' }} />
        <span className="group-name">{item.group.group_name}</span>
      </div>
    )
  }

  return null
}

// ── TmuxTree ──────────────────────────────────────────────────────────────

export function TmuxTree({
  sessions,
  groups = [],
  profileId,
  profileKey = '',
  onSelectPane,
  onRefresh,
  onOrderChange,
  onPaneContextMenu,
  onPaneStatusClick,
  statusRefreshToken: _statusRefreshToken,
  defaultExpanded = false,
}: Props) {
  const [sessionOrders, setSessionOrders] = useState<SessionOrder[]>([])
  const [groupOrders, setGroupOrders] = useState<{ id: number; sort_order: number }[]>([])
  const [activeItem, setActiveItem] = useState<TreeItem | null>(null)
  const [overItemId, setOverItemId] = useState<string | null>(null)
  const [statusMap, setStatusMap] = useState<Record<string, PaneStatus>>({})

  useEffect(() => {
    if (!profileId) {
      setSessionOrders([])
      setGroupOrders([])
      return
    }

    fetchProfileOrder(profileId).then((data) => {
      if (!data) return

      const orders: SessionOrder[] = []

      for (const g of data.groups || []) {
        for (const s of g.sessions || []) {
          orders.push({
            session_name: s.session_name,
            group_id: g.id,
            sort_order: s.sort_order,
          })
        }
      }

      for (const s of data.ungrouped || []) {
        orders.push({
          session_name: s.session_name,
          group_id: null,
          sort_order: s.sort_order,
        })
      }

      setSessionOrders(orders)

      setGroupOrders(
        (data.groups || []).map((g) => ({
          id: g.id,
          sort_order: g.sort_order,
        })),
      )
    })
  }, [profileId])

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  )

  const allPaneKeys = useMemo(() => {
    const keys: string[] = []
    sessions.forEach((session) => {
      session.windows.forEach((window) => {
        window.panes.forEach((pane) => {
          keys.push(buildPaneKey(session.sessionName, window.windowIndex, pane.paneId))
        })
      })
    })
    return keys
  }, [sessions])

  const buildStatusMap = useCallback(async () => {
    // Start with pane process statuses
    const paneItems =
      profileKey && allPaneKeys.length > 0 ? await fetchPaneStatuses(profileKey, allPaneKeys) : []
    const map: Record<string, PaneStatus> = {}
    paneItems.forEach((s) => {
      map[s.paneKey] = s.status
    })
    // Overlay with AI task statuses (take priority over idle)
    const taskMap = await fetchTaskPaneStatuses()
    for (const [key, status] of Object.entries(taskMap)) {
      if (!map[key] || map[key] === 'idle' || status === 'in_progress') {
        map[key] = status
      }
    }
    setStatusMap(map)
  }, [profileKey, allPaneKeys])

  useEffect(() => {
    buildStatusMap().catch(() => {})
  }, [buildStatusMap])

  // Auto-poll pane + task statuses every 10 seconds
  useEffect(() => {
    if (!profileKey && allPaneKeys.length === 0) return
    const interval = setInterval(() => {
      buildStatusMap().catch(() => {})
    }, 10000)
    return () => clearInterval(interval)
  }, [profileKey, allPaneKeys, buildStatusMap])

  const treeItems = useMemo(() => {
    const groupOrderMap = new Map(groupOrders.map((o) => [o.id, o.sort_order]))
    const sortedGroups = [...groups].sort((a, b) => {
      const aOrder = groupOrderMap.get(a.id) ?? a.sort_order
      const bOrder = groupOrderMap.get(b.id) ?? b.sort_order
      return aOrder - bOrder
    })
    const orderMap = new Map(sessionOrders.map((o) => [o.session_name, o]))

    const ungroupedSessions = sessions
      .filter((s) => {
        const order = orderMap.get(s.sessionName)
        return !order || order.group_id === null
      })
      .map((s, idx) => {
        const order = orderMap.get(s.sessionName)
        return {
          id: `session-${s.sessionName}`,
          type: 'session' as const,
          session: s,
          groupId: null,
          sortOrder: order?.sort_order ?? idx * 10,
        }
      })
      .sort((a, b) => a.sortOrder - b.sortOrder)

    const allRootItems: TreeItem[] = [
      ...ungroupedSessions,
      ...sortedGroups.map((g) => ({
        id: `group-${g.id}`,
        type: 'group' as const,
        group: g,
        groupId: null,
        sortOrder: groupOrderMap.get(g.id) ?? g.sort_order,
      })),
    ].sort((a, b) => a.sortOrder - b.sortOrder)

    return { rootItems: allRootItems, orderMap }
  }, [sessions, groups, sessionOrders, groupOrders])

  const getGroupSessions = useCallback(
    (groupId: number): TreeItem[] => {
      return sessions
        .filter((s) => {
          const order = treeItems.orderMap.get(s.sessionName)
          return order?.group_id === groupId
        })
        .map((s, idx) => {
          const order = treeItems.orderMap.get(s.sessionName)
          return {
            id: `session-${s.sessionName}`,
            type: 'session' as const,
            session: s,
            groupId,
            sortOrder: order?.sort_order ?? idx * 10,
          }
        })
        .sort((a, b) => a.sortOrder - b.sortOrder)
    },
    [sessions, treeItems.orderMap],
  )

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event
    const id = active.id as string

    let item = treeItems.rootItems.find((i) => i.id === id)
    if (!item) {
      for (const g of groups) {
        const groupSessions = getGroupSessions(g.id)
        item = groupSessions.find((i) => i.id === id)
        if (item) break
      }
    }

    setActiveItem(item || null)
  }

  const handleDragOver = (event: DragOverEvent) => {
    const { over } = event
    setOverItemId(over?.id as string | null)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event

    setActiveItem(null)
    setOverItemId(null)

    if (!over || active.id === over.id) return

    const activeId = active.id as string
    const overId = over.id as string

    const isActiveSession = activeId.startsWith('session-')
    const isActiveGroup = activeId.startsWith('group-')
    const isOverSession = overId.startsWith('session-')
    const isOverGroup = overId.startsWith('group-')

    let foundActiveItem = treeItems.rootItems.find((i) => i.id === activeId)
    if (!foundActiveItem) {
      for (const g of groups) {
        const groupSessions = getGroupSessions(g.id)
        foundActiveItem = groupSessions.find((i) => i.id === activeId)
        if (foundActiveItem) break
      }
    }

    let overItem = treeItems.rootItems.find((i) => i.id === overId)
    if (!overItem) {
      for (const g of groups) {
        const groupSessions = getGroupSessions(g.id)
        overItem = groupSessions.find((i) => i.id === overId)
        if (overItem) break
      }
    }

    if (!foundActiveItem || !overItem) return

    const newSessionOrders = [...sessionOrders]
    const newGroupOrders = [...groupOrders]
    const activeSessionName = foundActiveItem.session?.sessionName

    if (isActiveSession && activeSessionName) {
      let orderEntry = newSessionOrders.find((o) => o.session_name === activeSessionName)
      if (!orderEntry) {
        orderEntry = {
          session_name: activeSessionName,
          group_id: foundActiveItem.groupId,
          sort_order: 0,
        }
        newSessionOrders.push(orderEntry)
      }

      if (isOverGroup) {
        const targetGroupId = parseInt(overId.replace('group-', ''), 10)
        orderEntry.group_id = targetGroupId
        orderEntry.sort_order = 0
      } else if (isOverSession) {
        orderEntry.group_id = overItem.groupId
        orderEntry.sort_order =
          overItem.sortOrder + (foundActiveItem.sortOrder < overItem.sortOrder ? 1 : -1)
      }

      setSessionOrders(newSessionOrders)
    }

    if (isActiveGroup && foundActiveItem.group) {
      const activeGroupId = foundActiveItem.group.id
      let groupOrder = newGroupOrders.find((g) => g.id === activeGroupId)
      if (!groupOrder) {
        groupOrder = { id: activeGroupId, sort_order: foundActiveItem.sortOrder }
        newGroupOrders.push(groupOrder)
      }

      if (isOverGroup || isOverSession) {
        const overSortOrder = overItem.sortOrder
        groupOrder.sort_order = overSortOrder + (foundActiveItem.sortOrder < overSortOrder ? 1 : -1)
      }

      setGroupOrders(newGroupOrders)
    }

    const orderData: OrderData = {
      groups: groups.map((g) => {
        const order = newGroupOrders.find((o) => o.id === g.id)
        return {
          id: g.id,
          sort_order: order?.sort_order ?? g.sort_order,
        }
      }),
      sessions:
        newSessionOrders.length > 0
          ? newSessionOrders
          : sessions.map((s, idx) => ({
              session_name: s.sessionName,
              group_id: null,
              sort_order: idx * 10,
            })),
    }

    try {
      if (profileId !== undefined) {
        await saveOrder(profileId, orderData)
      }
      onOrderChange?.()
    } catch (_err) {}
  }

  const allSortableIds = useMemo(() => {
    const ids = treeItems.rootItems.map((i) => i.id)
    for (const g of groups) {
      const groupSessions = getGroupSessions(g.id)
      ids.push(...groupSessions.map((s) => s.id))
    }
    return ids
  }, [treeItems.rootItems, groups, getGroupSessions])

  const taskStats = useMemo(() => {
    const values = Object.values(statusMap)
    return {
      inProgress: values.filter((s) => s === 'in_progress').length,
      done: values.filter((s) => s === 'done').length,
      failed: values.filter((s) => s === 'failed').length,
      waiting: values.filter((s) => s === 'waiting').length,
      total: values.length,
    }
  }, [statusMap])

  const contextValue = useMemo(
    () => ({
      statusMap,
      actions: {
        onSelectPane,
        onRefresh,
        onPaneContextMenu,
        onPaneStatusClick,
        onGroupChanged: onOrderChange,
      },
      meta: {
        profileKey,
        groups,
        defaultExpanded,
      },
    }),
    [
      statusMap,
      onSelectPane,
      onRefresh,
      onPaneContextMenu,
      onPaneStatusClick,
      onOrderChange,
      profileKey,
      groups,
      defaultExpanded,
    ],
  )

  return (
    <div className="tmux-tree">
      <div className="tree-header">
        <span>Sessions</span>
        <div className="task-stats">
          {taskStats.inProgress > 0 && (
            <span className="task-stat task-stat--progress">
              <Loader2 size={10} className="spinning" />
              {taskStats.inProgress} 进行中
            </span>
          )}
          {taskStats.done > 0 && (
            <span className="task-stat task-stat--done">
              <CheckCircle2 size={10} />
              {taskStats.done} 已完成
            </span>
          )}
          {taskStats.failed > 0 && (
            <span className="task-stat task-stat--failed">
              <XCircle size={10} />
              {taskStats.failed} 失败
            </span>
          )}
          {taskStats.waiting > 0 && (
            <span className="task-stat task-stat--waiting">
              <Clock size={10} />
              {taskStats.waiting} 等待中
            </span>
          )}
        </div>
        <button type="button" onClick={onRefresh} className="refresh-btn" title="Refresh">
          <RefreshCw size={12} />
        </button>
        <NewTmuxButton sessions={sessions} onCreated={onRefresh} />
      </div>

      {sessions.length === 0 && groups.length === 0 && <div className="empty">No sessions</div>}

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragEnd={handleDragEnd}
      >
        <TmuxTreeContext.Provider value={contextValue}>
          <SortableContext items={allSortableIds} strategy={verticalListSortingStrategy}>
            <div className="tree-content">
              {treeItems.rootItems.map((item) => {
                if (item.type === 'group' && item.group) {
                  const groupSessions = getGroupSessions(item.group.id)
                  return (
                    <SortableGroup
                      key={item.id}
                      item={item}
                      group={item.group}
                      isOver={overItemId === item.id}
                    >
                      {groupSessions.length === 0 ? (
                        <div className="group-empty-drop">Drop sessions here</div>
                      ) : (
                        groupSessions.map((sessionItem) =>
                          sessionItem.session ? (
                            <SortableSession
                              key={sessionItem.id}
                              item={sessionItem}
                              session={sessionItem.session}
                              isInGroup={true}
                              isOver={overItemId === sessionItem.id}
                            />
                          ) : null,
                        )
                      )}
                    </SortableGroup>
                  )
                }

                if (item.type === 'session' && item.session) {
                  return (
                    <SortableSession
                      key={item.id}
                      item={item}
                      session={item.session}
                      isInGroup={false}
                      isOver={overItemId === item.id}
                    />
                  )
                }

                return null
              })}
            </div>
          </SortableContext>
        </TmuxTreeContext.Provider>

        <DragOverlay>
          <DragPreview item={activeItem} />
        </DragOverlay>
      </DndContext>
    </div>
  )
}
