import { createContext, useContext } from 'react'
import type { PaneStatus, SessionGroup } from '../../../types'

export interface TmuxTreeActions {
  onSelectPane: (paneId: string, paneName: string) => void
  onRefresh: () => void
  onPaneContextMenu?: (paneKey: string) => void
  onPaneStatusClick?: (paneKey: string) => void
  onGroupChanged?: () => void
}

export interface TmuxTreeMeta {
  profileKey: string
  groups: SessionGroup[]
  defaultExpanded: boolean
}

export interface TmuxTreeContextValue {
  statusMap: Record<string, PaneStatus>
  actions: TmuxTreeActions
  meta: TmuxTreeMeta
}

export const TmuxTreeContext = createContext<TmuxTreeContextValue | null>(null)

export function useTmuxTree(): TmuxTreeContextValue {
  const ctx = useContext(TmuxTreeContext)
  if (!ctx) throw new Error('useTmuxTree must be used within TmuxTreeContext')
  return ctx
}
