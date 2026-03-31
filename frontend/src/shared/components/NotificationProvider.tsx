import { createContext, type ReactNode, useCallback, useContext, useRef, useState } from 'react'

export interface Notification {
  id: string
  type: 'ai_completion' | 'ai_approval' | 'info' | 'error'
  title: string
  message: string
  paneKey: string
  timestamp: number
}

interface NotificationContextValue {
  notifications: Notification[]
  notify: (n: Omit<Notification, 'id' | 'timestamp'>) => void
  dismiss: (id: string) => void
}

const NotificationContext = createContext<NotificationContextValue | null>(null)

let nextId = 0

export function useNotification(): NotificationContextValue {
  const ctx = useContext(NotificationContext)
  if (!ctx) throw new Error('useNotification must be used within NotificationProvider')
  return ctx
}

interface NotificationProviderProps {
  children: ReactNode
  autoDismissMs?: number
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

function getNotificationStyle(type: Notification['type']): React.CSSProperties {
  return {
    padding: '12px 16px',
    borderRadius: 8,
    border: 'none',
    background:
      type === 'ai_completion'
        ? 'var(--accent-green, #9ece6a)'
        : type === 'ai_approval'
          ? 'var(--accent-yellow, #e0af68)'
          : 'var(--bg-tertiary, #24283b)',
    color:
      type === 'ai_completion' || type === 'ai_approval'
        ? '#1a1b26'
        : 'var(--text-primary, #c0caf5)',
    maxWidth: 400,
    boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
    cursor: 'pointer',
    textAlign: 'left',
    font: 'inherit',
  }
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

  return (
    <NotificationContext.Provider value={{ notifications, notify, dismiss }}>
      {children}
      <div style={CONTAINER_STYLE}>
        {notifications.map((n) => (
          <button
            key={n.id}
            type="button"
            aria-label={`Dismiss: ${n.title}`}
            style={getNotificationStyle(n.type)}
            onClick={() => dismiss(n.id)}
          >
            <div style={TITLE_STYLE}>{n.title}</div>
            <div style={MESSAGE_STYLE}>{n.message}</div>
          </button>
        ))}
      </div>
    </NotificationContext.Provider>
  )
}
