import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal } from '@xterm/xterm'
import type React from 'react'
import { useEffect, useRef, useState } from 'react'
import '@xterm/xterm/css/xterm.css'
import type { Metadata } from '../types'

interface TerminalTabProps {
  session: string
  pane: string
  onMetadata?: (metadata: Metadata) => void
  onTabRename?: (summary: string) => void
}

// Message types in gotty/webtty protocol:
// Client -> Server: '0'=Output (base64), '1'=Input, '2'=Ping, '3'=Resize, '4'=SetWindowTitle (unused)
// Server -> Client: '0'=Output (base64), '1'=Input (unused), '2'=Pong, '3'=SetWindowTitle

export function TerminalTab({ session, pane, onMetadata, onTabRename }: TerminalTabProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')
  const onMetadataRef = useRef(onMetadata)
  const onTabRenameRef = useRef(onTabRename)
  onMetadataRef.current = onMetadata
  onTabRenameRef.current = onTabRename

  useEffect(() => {
    if (!containerRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
      theme: {
        background: '#1a1b26',
        foreground: '#a9b1d6',
        cursor: '#c0caf5',
        cursorAccent: '#1a1b26',
        selectionBackground: '#33467c',
        black: '#15161e',
        red: '#f7768e',
        green: '#9ece6a',
        yellow: '#e0af68',
        blue: '#7aa2f7',
        magenta: '#bb9af7',
        cyan: '#7dcfff',
        white: '#a9b1d6',
        brightBlack: '#414868',
        brightRed: '#f7768e',
        brightGreen: '#9ece6a',
        brightYellow: '#e0af68',
        brightBlue: '#7aa2f7',
        brightMagenta: '#bb9af7',
        brightCyan: '#7dcfff',
        brightWhite: '#c0caf5',
      },
    })

    const fit = new FitAddon()
    term.loadAddon(fit)
    term.loadAddon(new WebLinksAddon())

    term.open(containerRef.current)
    fit.fit()

    termRef.current = term
    fitRef.current = fit

    // Build WebSocket URL with session/pane params
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = new URL(`${proto}//${location.host}/ws`)
    if (session) wsUrl.searchParams.set('session', session)
    if (pane) wsUrl.searchParams.set('pane', pane)

    const ws = new WebSocket(wsUrl.toString())
    wsRef.current = ws

    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      setStatus('connected')
      // Send auth init message (empty credentials if no auth)
      const initMsg = JSON.stringify({ AuthToken: '', Arguments: wsUrl.search.slice(1) })
      ws.send(initMsg)
      // Tell server to expect base64-encoded input
      ws.send('4base64')
    }

    ws.onmessage = (event) => {
      if (typeof event.data !== 'string') return
      const msgType = event.data[0]
      const payload = event.data.slice(1)

      switch (msgType) {
        case '1': // Output (base64 encoded)
          try {
            const decoded = atob(payload)
            term.write(decoded)
          } catch {
            // fallback: write raw
            term.write(payload)
          }
          break
        case '7': {
          // SetMetadata
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
        case '3': {
          // SetWindowTitle
          const title = JSON.parse(payload)
          if (title) {
            document.title = title
          }
          break
        }
      }
    }

    ws.onclose = () => {
      setStatus('disconnected')
      term.write('\r\n\x1b[33m[Connection closed]\x1b[0m\r\n')
    }

    ws.onerror = () => {
      setStatus('disconnected')
      term.write('\r\n\x1b[31m[Connection error]\x1b[0m\r\n')
    }

    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        // Send as type '1' (Input), base64 encoded
        const encoded = btoa(data)
        ws.send(`1${encoded}`)
      }
    })

    term.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(`3${JSON.stringify({ columns: cols, rows: rows })}`)
      }
    })

    const resizeObserver = new ResizeObserver(() => {
      fit.fit()
    })
    resizeObserver.observe(containerRef.current)

    return () => {
      resizeObserver.disconnect()
      ws.close()
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [session, pane])

  const statusColor =
    status === 'connected' ? '#9ece6a' : status === 'connecting' ? '#e0af68' : '#f7768e'

  return (
    <div style={styles.wrapper}>
      <div style={{ ...styles.statusBar, color: statusColor }}>
        ● {status}
        {session && (
          <span style={styles.sessionInfo}>
            {session}
            {pane ? `:${pane}` : ''}
          </span>
        )}
      </div>
      <div ref={containerRef} style={styles.terminal} />
    </div>
  )
}

const styles: Record<string, React.CSSProperties> = {
  wrapper: {
    display: 'flex',
    flexDirection: 'column',
    height: '100%',
    width: '100%',
  },
  statusBar: {
    height: '20px',
    fontSize: '11px',
    padding: '0 8px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    background: '#16161e',
    borderBottom: '1px solid #24283b',
    flexShrink: 0,
  },
  sessionInfo: {
    color: '#565f89',
  },
  terminal: {
    flex: 1,
    padding: '4px',
  },
}
