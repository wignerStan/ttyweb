import { useCallback, useEffect, useRef, useState } from 'react'
import type { Metadata } from '../types'

/**
 * Manages a WebSocket connection using the webtty binary protocol.
 *
 * @example
 * ```tsx
 * const { status, sendText, sendResize, wsRef } = useWebTTY({
 *   session: 'mysession',
 *   pane: '0',
 *   onOutput: (text) => terminal.write(text),
 *   reconnect: true,
 * })
 * ```
 */
export interface UseWebTTYOptions {
  session: string
  pane: string
  onOutput?: (text: string) => void
  onMetadata?: (metadata: Metadata) => void
  onTabRename?: (summary: string) => void
  onWindowTitle?: (title: string) => void
  onStatusChange?: (status: ConnectionStatus) => void
  onError?: () => void
  reconnect?: boolean
  maxReconnectAttempts?: number
}

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected'

/**
 * React hook for WebSocket communication via the webtty binary protocol.
 * Handles connection lifecycle, base64 encoding/decoding, and optional auto-reconnect.
 */
export function useWebTTY({
  session,
  pane,
  onOutput,
  onMetadata,
  onTabRename,
  onWindowTitle,
  onStatusChange,
  onError,
  reconnect = false,
  maxReconnectAttempts = 3,
}: UseWebTTYOptions) {
  const wsRef = useRef<WebSocket | null>(null)
  const [status, setStatus] = useState<ConnectionStatus>('connecting')
  const isCleanupRef = useRef(false)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)

  const onOutputRef = useRef(onOutput)
  const onMetadataRef = useRef(onMetadata)
  const onTabRenameRef = useRef(onTabRename)
  const onWindowTitleRef = useRef(onWindowTitle)
  const onStatusChangeRef = useRef(onStatusChange)
  onOutputRef.current = onOutput
  onMetadataRef.current = onMetadata
  onTabRenameRef.current = onTabRename
  onWindowTitleRef.current = onWindowTitle
  onStatusChangeRef.current = onStatusChange
  const onErrorRef = useRef(onError)
  onErrorRef.current = onError

  const buildWsUrl = useCallback(() => {
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = new URL(`${proto}//${location.host}/ws`)
    if (session) wsUrl.searchParams.set('session', session)
    if (pane) wsUrl.searchParams.set('pane', pane)
    return wsUrl
  }, [session, pane])

  const sendText = useCallback((text: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(`1${btoa(text)}`)
    }
  }, [])

  const sendResize = useCallback((cols: number, rows: number) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(`3${JSON.stringify({ columns: cols, rows })}`)
    }
  }, [])

  const sendRaw = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data)
    }
  }, [])

  const connect = useCallback(() => {
    if (isCleanupRef.current) return

    setStatus('connecting')
    onStatusChangeRef.current?.('connecting')

    const ws = new WebSocket(buildWsUrl().toString())
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      reconnectAttemptRef.current = 0
      setStatus('connected')
      onStatusChangeRef.current?.('connected')
      const wsUrl = buildWsUrl()
      const initMsg = JSON.stringify({ AuthToken: '', Arguments: wsUrl.search.slice(1) })
      ws.send(initMsg)
      ws.send('4base64')
    }

    ws.onmessage = (event) => {
      if (typeof event.data !== 'string') return
      const msgType = event.data[0]
      const payload = event.data.slice(1)

      switch (msgType) {
        case '1': {
          try {
            onOutputRef.current?.(atob(payload))
          } catch {
            onOutputRef.current?.(payload)
          }
          break
        }
        case '3': {
          try {
            const title = JSON.parse(payload)
            if (title) onWindowTitleRef.current?.(title)
          } catch {
            // ignore malformed title
          }
          break
        }
        case '7': {
          try {
            const metaJSON = atob(payload)
            const metadata: Metadata = JSON.parse(metaJSON)
            onMetadataRef.current?.(metadata)
            if (
              metadata.type === 'tab_rename' &&
              metadata.data?.summary &&
              onTabRenameRef.current
            ) {
              onTabRenameRef.current(String(metadata.data.summary))
            }
          } catch {
            // malformed metadata — ignore
          }
          break
        }
        default:
          // Unknown message type — ignore (protocol may add new types)
          break
      }
    }

    ws.onclose = () => {
      setStatus('disconnected')
      onStatusChangeRef.current?.('disconnected')
      if (isCleanupRef.current) return
      if (!reconnect) return

      reconnectAttemptRef.current += 1
      const attempt = reconnectAttemptRef.current
      if (attempt <= maxReconnectAttempts) {
        const delay = Math.min(1000 * 2 ** (attempt - 1), 16000)
        reconnectTimeoutRef.current = window.setTimeout(() => {
          if (!isCleanupRef.current) connect()
        }, delay)
      }
    }

    ws.onerror = () => {
      setStatus('disconnected')
      onStatusChangeRef.current?.('disconnected')
      onErrorRef.current?.()
    }

    wsRef.current = ws
  }, [buildWsUrl, reconnect, maxReconnectAttempts])

  useEffect(() => {
    isCleanupRef.current = false
    connect()
    return () => {
      isCleanupRef.current = true
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [connect])

  return { status, sendText, sendResize, sendRaw, connect, wsRef }
}
