import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from 'react'

export interface Notification {
  id: string
  type: 'ai_completion' | 'ai_approval' | 'info' | 'error'
  title: string
  message: string
  paneKey: string
  timestamp: number
}

// Stable context for notify/dismiss actions
interface NotificationActions {
  notify: (n: Omit<Notification, 'id' | 'timestamp'>) => void
  dismiss: (id: string) => void
}

// Full context for internal rendering
interface NotificationContextValue extends NotificationActions {
  notifications: Notification[]
}

const NotificationContext = createContext<NotificationContextValue | null>(null)
const NotificationActionsContext = createContext<NotificationActions | null>(null)

// Public hook — consumers that only send notifications use this.
// Gets the stable actions context and never re-renders on notification list changes.
export function useNotification(): NotificationActions {
  const ctx = useContext(NotificationActionsContext)
  if (!ctx) throw new Error('useNotification must be used within NotificationProvider')
  return ctx
}

// Internal hook — only for reading the notification list (rare, rendering only)
export function useNotificationList(): Notification[] {
  const ctx = useContext(NotificationContext)
  if (!ctx) throw new Error('useNotificationList must be used within NotificationProvider')
  return ctx.notifications
}

let nextId = 0

interface NotificationProviderProps {
  children: ReactNode
  autoDismissMs?: number
}

const NOTIFICATION_STYLES: Record<Notification['type'], React.CSSProperties> = {
  ai_completion: {
    padding: '12px 16px',
    borderRadius: 8,
    border: 'none',
    background: 'var(--accent-green, #9ece6a)',
    color: '#1a1b26',
    maxWidth: 400,
    boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
    cursor: 'pointer',
    textAlign: 'left',
    font: 'inherit',
  },
  ai_approval: {
    padding: '12px 16px',
    borderRadius: 8,
    border: 'none',
    background: 'var(--accent-yellow, #e0af68)',
    color: '#1a1b26',
    maxWidth: 400,
    boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
    cursor: 'pointer',
    textAlign: 'left',
    font: 'inherit',
  },
  info: {
    padding: '12px 16px',
    borderRadius: 8,
    border: 'none',
    background: 'var(--bg-tertiary, #24283b)',
    color: 'var(--text-primary, #c0caf5)',
    maxWidth: 400,
    boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
    cursor: 'pointer',
    textAlign: 'left',
    font: 'inherit',
  },
  error: {
    padding: '12px 16px',
    borderRadius: 8,
    border: 'none',
    background: 'var(--bg-tertiary, #24283b)',
    color: 'var(--text-primary, #c0caf5)',
    maxWidth: 400,
    boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
    cursor: 'pointer',
    textAlign: 'left',
    font: 'inherit',
  },
}

const CONTAINER_STYLE: React.CSSProperties = {
  position: 'fixed',
  top: 16,
  right: 16,
  zIndex: 9999,
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
}

const TITLE_STYLE: React.CSSProperties = { fontWeight: 600, fontSize: 14 }
const MESSAGE_STYLE: React.CSSProperties = { fontSize: 12, opacity: 0.9, marginTop: 4 }

export function NotificationProvider({
  children,
  autoDismissMs = 5000,
}: NotificationProviderProps) {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())

  const dismiss = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id))
    const timer = timersRef.current.get(id)
    if (timer) {
      clearTimeout(timer)
      timersRef.current.delete(id)
    }
  }, [])

  const notify = useCallback(
    (n: Omit<Notification, 'id' | 'timestamp'>) => {
      const id = `notif-${nextId++}`
      const notification: Notification = { ...n, id, timestamp: Date.now() }
      setNotifications((prev) => [...prev, notification])

      if (autoDismissMs > 0) {
        const timer = setTimeout(() => dismiss(id), autoDismissMs)
        timersRef.current.set(id, timer)
      }
    },
    [dismiss, autoDismissMs],
  )

  // Stable actions context — only changes when notify/dismiss identity changes
  const actions = useMemo(() => ({ notify, dismiss }), [notify, dismiss])

  // Full context for the notification renderer (internal only)
  const contextValue = useMemo(() => ({ ...actions, notifications }), [actions, notifications])

  return (
    <NotificationActionsContext.Provider value={actions}>
      <NotificationContext.Provider value={contextValue}>
        {children}
        <div style={CONTAINER_STYLE} aria-live="polite" aria-atomic="false" role="log">
          {notifications.map((n) => (
            <button
              key={n.id}
              type="button"
              aria-label={`Dismiss: ${n.title}`}
              style={NOTIFICATION_STYLES[n.type]}
              onClick={() => dismiss(n.id)}
            >
              <div style={TITLE_STYLE}>{n.title}</div>
              <div style={MESSAGE_STYLE}>{n.message}</div>
            </button>
          ))}
        </div>
      </NotificationContext.Provider>
    </NotificationActionsContext.Provider>
  )
}
