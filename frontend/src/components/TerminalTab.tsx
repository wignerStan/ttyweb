import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import { Terminal } from '@xterm/xterm'
import type React from 'react'
import { useCallback, useEffect, useRef } from 'react'
import '@xterm/xterm/css/xterm.css'
import { useWebTTY } from '../hooks/useWebTTY'
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
  const fitRef = useRef<FitAddon | null>(null)

  const onOutput = useCallback((text: string) => {
    termRef.current?.write(text)
  }, [])

  const onWindowTitle = useCallback((title: string) => {
    document.title = title
  }, [])

  const {
    status: wsStatus,
    sendText,
    sendResize,
  } = useWebTTY({
    session,
    pane,
    onOutput,
    onMetadata,
    onTabRename,
    onWindowTitle,
  })

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

    term.onData((data) => {
      sendText(data)
    })

    term.onResize(({ cols, rows }) => {
      sendResize(cols, rows)
    })

    const resizeObserver = new ResizeObserver(() => {
      fit.fit()
    })
    resizeObserver.observe(containerRef.current)

    return () => {
      resizeObserver.disconnect()
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [sendText, sendResize])

  const statusColor =
    wsStatus === 'connected' ? '#9ece6a' : wsStatus === 'connecting' ? '#e0af68' : '#f7768e'

  return (
    <div style={styles.wrapper}>
      <div style={{ ...styles.statusBar, color: statusColor }} aria-live="polite" role="status">
        ● {wsStatus}
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
