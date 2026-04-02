import type { PaneStatus, PaneStatusInfo } from '../../../types'
import { apiGet, apiPost, apiPut } from '../../../utils/api'
import { getAuthHeaders } from '../../../utils/auth'

// ── Types ──────────────────────────────────────────────────────────────────

export interface OrderData {
  groups: { id: number; sort_order: number }[]
  sessions: { session_name: string; group_id: number | null; sort_order: number }[]
}

export interface SessionOrder {
  session_name: string
  group_id: number | null
  sort_order: number
}

interface ProfileOrderResponse {
  groups: {
    id: number
    sort_order: number
    sessions: { session_name: string; sort_order: number }[]
  }[]
  ungrouped: { session_name: string; sort_order: number }[]
}

// ── Helpers ──────────────────────────────────────────────────────────────

/** Auth headers for raw fetch calls (e.g. rebuildSession which needs response status). */
function authFetchHeaders(): Record<string, string> {
  return { 'Content-Type': 'application/json', ...getAuthHeaders() }
}

// ── Tmux Window ───────────────────────────────────────────────────────────

export async function renameWindow(
  sessionName: string,
  windowIndex: number,
  newName: string,
): Promise<boolean> {
  const res = await apiPut<void>(
    `/api/tmux/windows/${encodeURIComponent(sessionName)}/${windowIndex}/rename`,
    { name: newName },
  )
  return res !== null
}

// ── Tmux Session Rebuild ─────────────────────────────────────────────────

export interface RebuildResult {
  ok: boolean
  message?: string
}

export async function rebuildSession(sessionName: string): Promise<RebuildResult> {
  try {
    const res = await fetch(`/api/tmux/sessions/${encodeURIComponent(sessionName)}/rebuild`, {
      method: 'POST',
      headers: authFetchHeaders(),
    })
    if (res.ok) return { ok: true }
    const data = await res.json().catch(() => ({}))
    return { ok: false, message: data.message || 'Unknown error' }
  } catch (err) {
    return { ok: false, message: err instanceof Error ? err.message : 'Network error' }
  }
}

// ── Profile Order ────────────────────────────────────────────────────────

export async function saveOrder(profileId: number, orderData: OrderData): Promise<void> {
  await apiPut<void>(`/api/profiles/${profileId}/order`, orderData)
}

export async function fetchProfileOrder(profileId: number): Promise<ProfileOrderResponse | null> {
  return apiGet<ProfileOrderResponse>(`/api/profiles/${profileId}/order`)
}

// ── Pane Statuses ────────────────────────────────────────────────────────

export async function fetchPaneStatuses(
  profileKey: string,
  paneKeys: string[],
): Promise<PaneStatusInfo[]> {
  if (!paneKeys.length) return []
  const encodedKeys = paneKeys.map((k) => encodeURIComponent(k)).join(',')
  const data = await apiGet<{ panes: PaneStatusInfo[] }>(
    `/api/panes/status?profile_key=${encodeURIComponent(profileKey)}&paneKeys=${encodedKeys}`,
  )
  return data?.panes ?? []
}

// ── Task Pane Statuses ───────────────────────────────────────────────────

interface TaskItem {
  pane_key: string
  task_status: string
}

interface TasksResponse {
  tasks: TaskItem[]
}

export async function fetchTaskPaneStatuses(): Promise<Record<string, PaneStatus>> {
  const data = await apiGet<TasksResponse>('/api/tasks?limit=500')
  if (!data) return {}
  const map: Record<string, PaneStatus> = {}
  const priority: Record<string, number> = {
    in_progress: 4,
    failed: 3,
    waiting: 2,
    done: 1,
    completed: 1,
  }
  for (const task of data.tasks || []) {
    const key: string = task.pane_key
    const status: string = task.task_status
    if (!key) continue
    const curP = priority[map[key] ?? ''] ?? 0
    const newP = priority[status ?? ''] ?? 0
    if (newP > curP) {
      map[key] = (status === 'completed' ? 'done' : status) as PaneStatus
    }
  }
  return map
}

// ── Session Group Assignment ─────────────────────────────────────────────

export async function assignSessionGroup(
  sessionName: string,
  profileKey: string,
  groupId: number | null,
): Promise<void> {
  await apiPut<void>(`/api/sessions/${encodeURIComponent(sessionName)}/group`, {
    profile_key: profileKey,
    group_id: groupId,
  })
}

export async function createGroup(profileKey: string, groupName: string): Promise<{ id?: number }> {
  const result = await apiPost<{ id?: number }>('/api/groups', {
    profile_key: profileKey,
    group_name: groupName,
  })
  return result ?? {}
}
