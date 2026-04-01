// WorkerContextMenu.tsx — Right-click context menu for WorkerCard

import { Copy, Pause, Power, Terminal } from 'lucide-react'
import { useEffect, useRef } from 'react'
import { getAuthHeaders } from '../../../../utils/auth'
import { BUTLER_API_BASE } from '../constants'

interface WorkerContextMenuProps {
  x: number
  y: number
  workerId: string
  paneTarget: string
  onClose: () => void
}

function buildHeaders(contentType?: string): Record<string, string> {
  const headers: Record<string, string> = { ...getAuthHeaders() }
  if (contentType) headers['Content-Type'] = contentType
  return headers
}

export function WorkerContextMenu({ x, y, workerId, paneTarget, onClose }: WorkerContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose()
      }
    }
    const onEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    const t = setTimeout(() => {
      document.addEventListener('click', onClickOutside)
      document.addEventListener('keydown', onEscape)
    }, 10)
    return () => {
      clearTimeout(t)
      document.removeEventListener('click', onClickOutside)
      document.removeEventListener('keydown', onEscape)
    }
  }, [onClose])

  const handle = (action: 'open' | 'copy' | 'pause' | 'kill') => {
    switch (action) {
      case 'open':
        window.dispatchEvent(new CustomEvent('imperial:focus-pane', { detail: { paneTarget } }))
        break
      case 'copy':
        navigator.clipboard.writeText(paneTarget).catch((err) => {
          // biome-ignore lint/suspicious/noConsole: clipboard failure should be visible
          console.error('Failed to copy:', err)
        })
        break
      case 'pause':
        fetch(`${BUTLER_API_BASE}/worker_sessions/${workerId}`, {
          method: 'PUT',
          headers: buildHeaders('application/json'),
          body: JSON.stringify({ state: 'paused' }),
        }).catch((_err) => {})
        break
      case 'kill':
        if (!confirm('Kill this worker?')) return
        fetch(`${BUTLER_API_BASE}/worker_sessions/${workerId}`, {
          method: 'DELETE',
          headers: buildHeaders(),
        }).catch((_err) => {})
        break
    }
    onClose()
  }

  return (
    <div ref={menuRef} className="is-ctx-menu" style={{ left: x, top: y }}>
      <button
        type="button"
        className="is-ctx-menu__item"
        style={{
          background: 'none',
          border: 'none',
          padding: 0,
          font: 'inherit',
          color: 'inherit',
          cursor: 'pointer',
          width: '100%',
          textAlign: 'inherit',
        }}
        onClick={() => handle('open')}
      >
        <Terminal size={14} />
        <span>Open Terminal</span>
      </button>
      <button
        type="button"
        className="is-ctx-menu__item"
        style={{
          background: 'none',
          border: 'none',
          padding: 0,
          font: 'inherit',
          color: 'inherit',
          cursor: 'pointer',
          width: '100%',
          textAlign: 'inherit',
        }}
        onClick={() => handle('copy')}
      >
        <Copy size={14} />
        <span>Copy pane target</span>
      </button>
      <button
        type="button"
        className="is-ctx-menu__item"
        style={{
          background: 'none',
          border: 'none',
          padding: 0,
          font: 'inherit',
          color: 'inherit',
          cursor: 'pointer',
          width: '100%',
          textAlign: 'inherit',
        }}
        onClick={() => handle('pause')}
      >
        <Pause size={14} />
        <span>Pause worker</span>
      </button>
      <button
        type="button"
        className="is-ctx-menu__item danger"
        style={{
          background: 'none',
          border: 'none',
          padding: 0,
          font: 'inherit',
          color: 'inherit',
          cursor: 'pointer',
          width: '100%',
          textAlign: 'inherit',
        }}
        onClick={() => handle('kill')}
      >
        <Power size={14} />
        <span>Kill worker</span>
      </button>
    </div>
  )
}
