// constants.ts — Color maps, polling intervals, icon maps
import {
  AlertTriangle,
  CheckCircle2,
  FileText,
  HelpCircle,
  type LucideIcon,
  ShieldCheck,
} from 'lucide-react'
import type { InboxKind, WorkerState } from './types'

// ── Worker State Colors ──────────────────────────────────────────────────────
export const WORKER_STATE_DOT_COLOR: Record<WorkerState, string> = {
  launching: 'var(--color-warning)',
  idle: 'var(--color-warning)',
  busy: 'var(--color-success)',
  exited: 'var(--color-on-surface-muted)',
  error: 'var(--color-error)',
}

export const WORKER_STATE_LABEL_COLOR: Record<WorkerState, string> = {
  launching: 'var(--color-warning)',
  idle: 'var(--color-on-surface-muted)',
  busy: 'var(--color-success)',
  exited: 'var(--color-on-surface-muted)',
  error: 'var(--color-error)',
}

// ── Inbox Kind → Icon + Color ────────────────────────────────────────────────
export const INBOX_KIND_CONFIG: Record<InboxKind, { icon: LucideIcon; color: string }> = {
  question: { icon: HelpCircle, color: 'var(--color-primary)' },
  approval: { icon: ShieldCheck, color: 'var(--color-warning)' },
  report: { icon: FileText, color: 'var(--color-on-surface-muted)' },
  error: { icon: AlertTriangle, color: 'var(--color-error)' },
  completion: { icon: CheckCircle2, color: 'var(--color-success)' },
}

// ── Default fallback icon and color for unknown inbox kinds ──────────────────
export const INBOX_KIND_DEFAULT_ICON = FileText
export const INBOX_KIND_DEFAULT_COLOR = 'var(--color-on-surface-muted)'

// ── Polling Intervals (ms) ───────────────────────────────────────────────────
export const POLL_WORKERS_MS = 5_000
export const POLL_INBOX_MS = 5_000
export const POLL_ACTIVITY_MS = 10_000

// ── Butler API Base URL (proxied by TmuxWeb) ─────────────────────────────────
export const BUTLER_API_BASE = '/api/butler'
