import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import { useCallback, useEffect, useRef, useState } from 'react'
import '@xterm/xterm/css/xterm.css'
import { Maximize2 } from 'lucide-react'
import type { VoiceInputHandle } from '../shared/components/VoiceInput'
import { isAndroid, isIOS } from '../utils/platform'
import { log as telemetryLog } from '../utils/telemetry'
import { createTelemetryEmitter, type TelemetryEmitter } from '../utils/telemetryEmitter'
import { MobileToolbox } from './MobileToolbox'

const DEC_1004_DISABLE = '\x1b[?1004l'
const LONG_PRESS_MS = 650
const LONG_PRESS_MOVE_TOLERANCE = 10
const BURST_SUPPRESSION_WINDOW_MS = 200
const SUPPRESSED_INPUTS = new Set([' ', '\r', '\n'])
const SPACE_BURST_COUNT = 3
const SPACE_BURST_WINDOW_MS = 500
const ENTER_BURST_COUNT = 2
const ENTER_BURST_WINDOW_MS = 500
const SCROLL_THRESHOLD = 20
const MAX_RECONNECT_ATTEMPTS = 3

const TERMINAL_THEME = {
  background: '#0f1115',
  foreground: '#abb2bf',
  cursor: '#4d78cc',
  selectionBackground: 'rgba(77, 120, 204, 0.3)',
  black: '#1e2127',
  red: '#e06c75',
  green: '#98c379',
  yellow: '#d19a66',
  blue: '#61afef',
  magenta: '#c678dd',
  cyan: '#56b6c2',
  white: '#abb2bf',
}

interface MobileTerminalProps {
  session: string
  pane: string
  fontSize: number
  onFontSizeChange: (size: number) => void
  voiceRef?: React.RefObject<VoiceInputHandle | null>
  taskHistoryPaneKey?: string | null
  onStatusChange?: () => void
}

export function MobileTerminal({
  session,
  pane,
  fontSize,
  onFontSizeChange,
  voiceRef,
  taskHistoryPaneKey,
  onStatusChange,
}: MobileTerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const reconnectAttemptRef = useRef(0)
  const isCleanupRef = useRef(false)
  const lastTransitionRef = useRef<{
    type: 'reconnect' | 'visibility' | 'keyboard'
    time: number
  } | null>(null)
  const emitterRef = useRef<TelemetryEmitter | null>(null)
  const intentionalCloseRef = useRef(false)
  const manualReconnectDisposable = useRef<{ dispose: () => void } | null>(null)
  const [showKeyboard, setShowKeyboard] = useState(false)
  const selectOverlayRef = useRef<HTMLDivElement | null>(null)

  const sendText = useCallback((text: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const encoded = btoa(text)
      wsRef.current.send(`1${encoded}`)
    }
  }, [])

  const toggleKeyboard = useCallback(() => {
    setShowKeyboard((prev) => {
      const next = !prev
      if (next) {
        const textarea = document.querySelector(
          '.xterm-helper-textarea',
        ) as HTMLTextAreaElement | null
        textarea?.focus()
      }
      return next
    })
  }, [])

  // Update terminal font size when prop changes
  useEffect(() => {
    if (termRef.current && fitRef.current) {
      termRef.current.options.fontSize = fontSize
      fitRef.current.fit()
      setTimeout(() => fitRef.current?.fit(), 100)
      setTimeout(() => {
        if (!fitRef.current || !termRef.current) return
        fitRef.current.fit()
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(
            `3${JSON.stringify({ columns: termRef.current.cols, rows: termRef.current.rows })}`,
          )
        }
      }, 300)
    }
  }, [fontSize])

  useEffect(() => {
    if (!containerRef.current) return
    isCleanupRef.current = false
    intentionalCloseRef.current = false

    const paneId = `${session}:${pane}`
    const emitter = createTelemetryEmitter(paneId)
    emitterRef.current = emitter

    const term = new Terminal({
      cursorBlink: true,
      fontSize,
      fontFamily: 'Menlo, Monaco, monospace',
      theme: TERMINAL_THEME,
      scrollback: 5000,
      lineHeight: 1.2,
      drawBoldTextInBrightColors: true,
      cursorStyle: 'bar',
    })

    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(containerRef.current)
    // iOS fit retries
    fit.fit()
    setTimeout(() => fit.fit(), 100)
    setTimeout(() => fit.fit(), 300)

    const textarea = containerRef.current.querySelector('textarea')
    if (textarea) {
      textarea.setAttribute('autocapitalize', 'off')
      textarea.setAttribute('autocorrect', 'off')
      textarea.setAttribute('spellcheck', 'false')
      textarea.setAttribute('autocomplete', 'off')
    }

    termRef.current = term
    fitRef.current = fit

    let paneInAltScreen = false

    const checkPaneMode = () => {
      fetch(`/api/tmux/pane-mode?paneId=${encodeURIComponent(pane)}`)
        .then((r) => r.json())
        .then((data) => {
          paneInAltScreen = !!(data.alternate_on && data.mouse_any_flag)
        })
        .catch(() => {})
    }
    checkPaneMode()
    const paneModeInterval = setInterval(checkPaneMode, 5000)

    // --- Touch gesture system ---
    const sendScroll = (lines: number) => {
      if (lines === 0 || wsRef.current?.readyState !== WebSocket.OPEN) return
      const count = Math.min(Math.abs(lines), 10)
      const cols = termRef.current?.cols ?? 80
      const rows = termRef.current?.rows ?? 24
      const cx = Math.floor(cols / 2)
      const cy = Math.floor(rows / 2)
      const button = lines > 0 ? 65 : 64
      for (let i = 0; i < count; i++) {
        wsRef.current?.send(`\x1b[<${button};${cx};${cy}M`)
      }
    }

    type GestureState = 'idle' | 'oneFinger' | 'twoFingerScroll'
    let gesture: GestureState = 'idle'
    let twoFingerStartY = 0
    let twoFingerAccum = 0
    let clickBlockedUntil = 0
    let longPressTimer: number | null = null
    let longPressStartX = 0
    let longPressStartY = 0

    const cancelLongPress = () => {
      if (longPressTimer !== null) {
        clearTimeout(longPressTimer)
        longPressTimer = null
      }
    }

    const showSelectionOverlay = () => {
      const container = containerRef.current
      const currentTerm = termRef.current
      if (!container || !currentTerm) return

      const buf = currentTerm.buffer.active
      const bufLines: string[] = []
      const start = Math.max(0, buf.viewportY)
      const end = start + currentTerm.rows
      for (let i = start; i < end; i++) {
        const line = buf.getLine(i)
        if (line) bufLines.push(line.translateToString(true))
      }
      while (bufLines.length > 0 && bufLines[bufLines.length - 1]?.trim() === '') bufLines.pop()
      const text = bufLines.join('\n')
      if (!text) return

      const overlay = document.createElement('div')
      overlay.className = 'select-mode-overlay'

      const toolbar = document.createElement('div')
      toolbar.className = 'select-mode-toolbar'
      const copyBtn = document.createElement('button')
      copyBtn.className = 'select-mode-btn select-mode-copy-btn'
      copyBtn.textContent = '\u590D\u5236\u5E76\u9000\u51FA'
      const exitBtn = document.createElement('button')
      exitBtn.className = 'select-mode-btn select-mode-exit-btn'
      exitBtn.textContent = '\u9000\u51FA'
      toolbar.appendChild(copyBtn)
      toolbar.appendChild(exitBtn)
      overlay.appendChild(toolbar)

      const textArea = document.createElement('pre')
      textArea.className = 'select-mode-text'
      textArea.textContent = text
      overlay.appendChild(textArea)

      copyBtn.addEventListener('touchend', async (ev) => {
        ev.preventDefault()
        ev.stopPropagation()
        const sel = window.getSelection()?.toString()
        if (sel) {
          try {
            await navigator.clipboard.writeText(sel)
          } catch {
            const ta = document.createElement('textarea')
            ta.value = sel
            ta.style.cssText = 'position:fixed;left:-9999px'
            document.body.appendChild(ta)
            ta.select()
            document.execCommand('copy')
            document.body.removeChild(ta)
          }
          copyBtn.textContent = '\u5DF2\u590D\u5236 \u2713'
          copyBtn.classList.add('select-mode-copied')
          setTimeout(() => hideSelectionOverlay(), 2000)
          return
        }
        hideSelectionOverlay()
      })

      exitBtn.addEventListener('touchend', (ev) => {
        ev.preventDefault()
        ev.stopPropagation()
        hideSelectionOverlay()
      })

      container.style.position = 'relative'
      container.appendChild(overlay)
      selectOverlayRef.current = overlay
    }

    const hideSelectionOverlay = () => {
      const overlay = selectOverlayRef.current
      if (overlay?.parentNode) {
        overlay.parentNode.removeChild(overlay)
      }
      selectOverlayRef.current = null
      window.getSelection()?.removeAllRanges()
    }

    const onTouchStart = (e: TouchEvent) => {
      const prevGesture = gesture
      if (e.touches.length === 2) {
        gesture = 'twoFingerScroll'
        cancelLongPress()
        e.preventDefault()
        e.stopPropagation()
        twoFingerStartY = ((e.touches[0]?.clientY ?? 0) + (e.touches[1]?.clientY ?? 0)) / 2
        twoFingerAccum = 0
      } else if (e.touches.length === 1 && gesture === 'idle') {
        gesture = 'oneFinger'
        longPressStartX = e.touches[0]?.clientX ?? 0
        longPressStartY = e.touches[0]?.clientY ?? 0
        longPressTimer = window.setTimeout(() => {
          longPressTimer = null
          showSelectionOverlay()
        }, LONG_PRESS_MS)
      }
      emitter.emit('touch-start', {
        touches: e.touches.length,
        prevGesture,
        newGesture: gesture,
        prevented: e.touches.length === 2,
        y0: e.touches[0]?.clientY,
        y1: e.touches[1]?.clientY,
      })
    }

    const onTouchMove = (e: TouchEvent) => {
      if (longPressTimer !== null && e.touches.length === 1) {
        const dx = (e.touches[0]?.clientX ?? 0) - longPressStartX
        const dy = (e.touches[0]?.clientY ?? 0) - longPressStartY
        if (Math.sqrt(dx * dx + dy * dy) > LONG_PRESS_MOVE_TOLERANCE) {
          cancelLongPress()
        }
      }
      if (gesture === 'oneFinger' && e.touches.length === 2) {
        gesture = 'twoFingerScroll'
        cancelLongPress()
        e.preventDefault()
        e.stopPropagation()
        twoFingerStartY = ((e.touches[0]?.clientY ?? 0) + (e.touches[1]?.clientY ?? 0)) / 2
        twoFingerAccum = 0
        emitter.emit('touch-move', { upgrade: true, from: 'oneFinger', touches: 2 })
        return
      }
      if (gesture === 'twoFingerScroll' && e.touches.length >= 2) {
        e.preventDefault()
        e.stopPropagation()
        const midY = ((e.touches[0]?.clientY ?? 0) + (e.touches[1]?.clientY ?? 0)) / 2
        const deltaY = twoFingerStartY - midY
        twoFingerAccum += deltaY
        twoFingerStartY = midY
        const lines = Math.trunc(twoFingerAccum / SCROLL_THRESHOLD)
        if (lines !== 0) {
          twoFingerAccum -= lines * SCROLL_THRESHOLD
          sendScroll(lines)
          emitter.emit('touch-scroll', {
            lines,
            deltaY: Math.round(deltaY),
            accum: Math.round(twoFingerAccum),
            altScreen: paneInAltScreen,
          })
        }
      }
    }

    const onTouchEnd = (e: TouchEvent) => {
      cancelLongPress()
      const prevGesture = gesture
      if (gesture === 'twoFingerScroll') {
        e.preventDefault()
        e.stopPropagation()
        if (e.touches.length === 0) {
          clickBlockedUntil = Date.now() + 300
          gesture = 'idle'
        }
      } else if (gesture === 'oneFinger' && e.touches.length === 0) {
        gesture = 'idle'
      }
      emitter.emit('touch-end', {
        prevGesture,
        newGesture: gesture,
        remainingTouches: e.touches.length,
        prevented: prevGesture === 'twoFingerScroll',
      })
    }

    const onClickBlock = (e: MouseEvent) => {
      if (Date.now() < clickBlockedUntil) {
        e.preventDefault()
        e.stopPropagation()
        emitter.emit('touch-click-blocked', { ttl: clickBlockedUntil - Date.now() })
      }
    }

    const termContainer = containerRef.current
    const xtermScreen = termContainer.querySelector('.xterm-screen') as HTMLElement | null
    const touchTarget = xtermScreen || termContainer
    touchTarget.addEventListener('touchstart', onTouchStart, { capture: true, passive: false })
    touchTarget.addEventListener('touchmove', onTouchMove, { capture: true, passive: false })
    touchTarget.addEventListener('touchend', onTouchEnd, { capture: true, passive: false })
    touchTarget.addEventListener('touchcancel', onTouchEnd, { capture: true, passive: false })
    touchTarget.addEventListener('click', onClickBlock, { capture: true })
    emitter.emit('touch-gesture-info', {
      msg: 'gesture-listeners-bound',
      targetTag: touchTarget.tagName,
      targetClass: touchTarget.className,
      isXtermScreen: !!xtermScreen,
    })

    // --- WebSocket connection (webtty protocol) ---
    const buildWsUrl = () => {
      const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = new URL(`${proto}//${location.host}/ws`)
      if (session) wsUrl.searchParams.set('session', session)
      if (pane) wsUrl.searchParams.set('pane', pane)
      return wsUrl
    }

    const connect = () => {
      if (isCleanupRef.current) return
      manualReconnectDisposable.current?.dispose()
      manualReconnectDisposable.current = null

      const ws = new WebSocket(buildWsUrl().toString())
      ws.binaryType = 'arraybuffer'
      ws.onopen = () => {
        const wasReconnect = reconnectAttemptRef.current > 0
        reconnectAttemptRef.current = 0

        // webtty handshake: AuthToken + base64 encoding preference
        const wsUrl = buildWsUrl()
        const initMsg = JSON.stringify({ AuthToken: '', Arguments: wsUrl.search.slice(1) })
        ws.send(initMsg)
        ws.send('4base64')

        if (termRef.current) {
          ws.send(
            `3${JSON.stringify({ columns: termRef.current.cols, rows: termRef.current.rows })}`,
          )
        }

        if (wasReconnect) {
          termRef.current?.write('\r\x1b[2K\x1b[32m[\u5DF2\u91CD\u8FDE]\x1b[0m\r\n')
        }

        if (isIOS()) {
          ws.send(DEC_1004_DISABLE)
          telemetryLog('dec1004-disable', { trigger: 'onopen' })
          if (wasReconnect) {
            lastTransitionRef.current = { type: 'reconnect', time: Date.now() }
            telemetryLog('reconnect', { timestamp: Date.now() })
            emitter.emit('mobile-transition', { kind: 'reconnect' })
          }
        }
      }

      ws.onmessage = (event) => {
        if (typeof event.data !== 'string') return
        const msgType = event.data[0]
        const payload = event.data.slice(1)

        switch (msgType) {
          case '1': // Output (base64 encoded)
            try {
              termRef.current?.write(atob(payload))
            } catch {
              termRef.current?.write(payload)
            }
            break
          case '3': // SetWindowTitle
            try {
              const title = JSON.parse(payload)
              if (title) document.title = title
            } catch {
              // ignore malformed title
            }
            break
        }
      }

      ws.onerror = () => {
        termRef.current?.write('\r\n\x1b[33m[Connection error]\x1b[0m\r\n')
      }

      // Exponential backoff reconnection
      ws.onclose = () => {
        if (isCleanupRef.current || intentionalCloseRef.current) return
        reconnectAttemptRef.current += 1
        const attempt = reconnectAttemptRef.current
        if (attempt <= MAX_RECONNECT_ATTEMPTS) {
          const delay = Math.min(1000 * 2 ** (attempt - 1), 16000)
          termRef.current?.write(
            `\r\n\x1b[33m[\u8FDE\u63A5\u65AD\u5F00\uFF0C${Math.round(delay / 1000)}s \u540E\u91CD\u8FDE (${attempt}/${MAX_RECONNECT_ATTEMPTS})...]\x1b[0m`,
          )
          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (!isCleanupRef.current && !intentionalCloseRef.current) connect()
          }, delay)
        } else {
          termRef.current?.write(
            '\r\n\x1b[31m[\u91CD\u8FDE\u5931\u8D25]\x1b[0m \x1b[33m\u6309\u4EFB\u610F\u952E\u91CD\u8FDE\uFF0C\u6216\u5173\u95ED\u91CD\u65B0\u6253\u5F00\x1b[0m\r\n',
          )
          if (termRef.current) {
            manualReconnectDisposable.current = termRef.current.onData(() => {
              manualReconnectDisposable.current?.dispose()
              manualReconnectDisposable.current = null
              reconnectAttemptRef.current = 0
              connect()
            })
          }
        }
      }

      wsRef.current = ws
    }

    // Visibility change handler — reconnect on resume
    const handleVisibilityChange = () => {
      telemetryLog('visibilitychange', { state: document.visibilityState })
      emitter.emit('mobile-transition', { kind: 'visibility', state: document.visibilityState })

      if (document.visibilityState === 'visible') {
        if (isIOS()) {
          lastTransitionRef.current = { type: 'visibility', time: Date.now() }
        }
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current)
          reconnectTimeoutRef.current = null
        }
        if (
          wsRef.current?.readyState !== WebSocket.OPEN &&
          wsRef.current?.readyState !== WebSocket.CONNECTING
        ) {
          termRef.current?.write('\r\n\x1b[36m[Resuming connection...]\x1b[0m\r\n')
          reconnectAttemptRef.current = 0
          connect()
        }
      }
    }
    document.addEventListener('visibilitychange', handleVisibilityChange)

    // --- Burst suppression (iOS/Android) ---
    const spaceTimestamps: number[] = []
    const enterTimestamps: number[] = []

    const shouldSuppressBurstIOS = (data: string, now: number): boolean => {
      if (data === ' ') {
        spaceTimestamps.push(now)
        while (
          spaceTimestamps.length > 0 &&
          now - (spaceTimestamps[0] ?? 0) > SPACE_BURST_WINDOW_MS
        ) {
          spaceTimestamps.shift()
        }
        if (spaceTimestamps.length >= SPACE_BURST_COUNT) {
          telemetryLog('suppressed', {
            data: JSON.stringify(data),
            reason: 'space-burst',
            count: spaceTimestamps.length,
          })
          emitter.emit('mobile-suppress', {
            reason: 'space-burst',
            data: JSON.stringify(data),
            count: spaceTimestamps.length,
          })
          spaceTimestamps.length = 0
          return true
        }
      }
      if (data === '\r' || data === '\n') {
        enterTimestamps.push(now)
        while (
          enterTimestamps.length > 0 &&
          now - (enterTimestamps[0] ?? 0) > ENTER_BURST_WINDOW_MS
        ) {
          enterTimestamps.shift()
        }
        if (enterTimestamps.length >= ENTER_BURST_COUNT) {
          telemetryLog('suppressed', {
            data: JSON.stringify(data),
            reason: 'enter-burst',
            count: enterTimestamps.length,
          })
          emitter.emit('mobile-suppress', {
            reason: 'enter-burst',
            data: JSON.stringify(data),
            count: enterTimestamps.length,
          })
          enterTimestamps.length = 0
          return true
        }
      }
      // iOS: post-transition suppression
      if (!SUPPRESSED_INPUTS.has(data)) return false
      const transition = lastTransitionRef.current
      if (!transition) return false
      const elapsed = now - transition.time
      if (elapsed < BURST_SUPPRESSION_WINDOW_MS) {
        telemetryLog('suppressed', {
          data: JSON.stringify(data),
          transitionType: transition.type,
          elapsed,
        })
        emitter.emit('mobile-suppress', {
          reason: 'post-transition',
          data: JSON.stringify(data),
          transitionType: transition.type,
          elapsed,
        })
        return true
      }
      return false
    }

    const shouldSuppressBurstAndroid = (data: string, now: number): boolean => {
      if (data === ' ') {
        spaceTimestamps.push(now)
        while (
          spaceTimestamps.length > 0 &&
          now - (spaceTimestamps[0] ?? 0) > SPACE_BURST_WINDOW_MS
        ) {
          spaceTimestamps.shift()
        }
        if (spaceTimestamps.length >= SPACE_BURST_COUNT) {
          telemetryLog('suppressed', {
            data: JSON.stringify(data),
            reason: 'space-burst',
            count: spaceTimestamps.length,
          })
          emitter.emit('mobile-suppress', {
            reason: 'space-burst',
            data: JSON.stringify(data),
            count: spaceTimestamps.length,
          })
          spaceTimestamps.length = 0
          return true
        }
      }
      if (data === '\r' || data === '\n') {
        enterTimestamps.push(now)
        while (
          enterTimestamps.length > 0 &&
          now - (enterTimestamps[0] ?? 0) > ENTER_BURST_WINDOW_MS
        ) {
          enterTimestamps.shift()
        }
        if (enterTimestamps.length >= ENTER_BURST_COUNT) {
          telemetryLog('suppressed', {
            data: JSON.stringify(data),
            reason: 'enter-burst',
            count: enterTimestamps.length,
          })
          emitter.emit('mobile-suppress', {
            reason: 'enter-burst',
            data: JSON.stringify(data),
            count: enterTimestamps.length,
          })
          enterTimestamps.length = 0
          return true
        }
      }
      return false
    }

    const shouldSuppressBurst = (data: string): boolean => {
      const now = Date.now()
      if (isIOS()) return shouldSuppressBurstIOS(data, now)
      if (isAndroid()) return shouldSuppressBurstAndroid(data, now)
      return false
    }

    // --- Terminal input → WebSocket (webtty: base64-encoded) ---
    let lastInputData = ''
    let lastInputTime = 0

    term.onData((data) => {
      if (wsRef.current?.readyState !== WebSocket.OPEN) return

      // Filter out focus reports and device attributes
      if (
        data === '\x1b[I' ||
        data === '\x1b[O' ||
        (data.startsWith('\x1b[?') && data.endsWith('c')) ||
        (data.startsWith('\x1b[>') && data.endsWith('c')) ||
        data.startsWith('\x1b]')
      ) {
        return
      }

      if (shouldSuppressBurst(data)) return

      const now = Date.now()
      if (data === lastInputData && now - lastInputTime < 50) return
      lastInputData = data
      lastInputTime = now

      telemetryLog('onData', { data: JSON.stringify(data), len: data.length })
      emitter.emit('mobile-onData', {
        data: JSON.stringify(data),
        len: data.length,
        wsReadyState: wsRef.current?.readyState,
      })

      // webtty protocol: type '1' (Input) + base64 encoded
      const encoded = btoa(data)
      wsRef.current.send(`1${encoded}`)
    })

    connect()

    // --- Visual viewport tracking (iOS keyboard) ---
    let viewportCleanup: (() => void) | undefined
    if (isIOS() && window.visualViewport) {
      const handleViewportResize = () => {
        lastTransitionRef.current = { type: 'keyboard', time: Date.now() }
        telemetryLog('viewport-resize', {
          height: window.visualViewport?.height,
          width: window.visualViewport?.width,
        })
        emitter.emit('mobile-transition', {
          kind: 'keyboard',
          height: window.visualViewport?.height,
          width: window.visualViewport?.width,
        })
      }
      window.visualViewport.addEventListener('resize', handleViewportResize)
      viewportCleanup = () =>
        window.visualViewport?.removeEventListener('resize', handleViewportResize)
    }

    // --- Resize observer ---
    let lastCols = 0
    let lastRows = 0

    const handleResize = () => {
      if (!fitRef.current || !termRef.current) return
      fitRef.current.fit()
      const cols = termRef.current.cols
      const rows = termRef.current.rows
      if (cols !== lastCols || rows !== lastRows) {
        lastCols = cols
        lastRows = rows
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(`3${JSON.stringify({ columns: cols, rows: rows })}`)
        }
      }
    }

    let resizeTimeout: number | null = null
    const resizeObserver = new ResizeObserver(() => {
      if (resizeTimeout) clearTimeout(resizeTimeout)
      resizeTimeout = window.setTimeout(handleResize, 150)
    })
    resizeObserver.observe(containerRef.current)

    return () => {
      isCleanupRef.current = true
      intentionalCloseRef.current = true
      emitter.destroy()
      emitterRef.current = null
      manualReconnectDisposable.current?.dispose()
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      viewportCleanup?.()
      clearInterval(paneModeInterval)
      touchTarget.removeEventListener('touchstart', onTouchStart, true)
      touchTarget.removeEventListener('touchmove', onTouchMove, true)
      touchTarget.removeEventListener('touchend', onTouchEnd, true)
      touchTarget.removeEventListener('touchcancel', onTouchEnd, true)
      touchTarget.removeEventListener('click', onClickBlock, true)
      cancelLongPress()
      hideSelectionOverlay()
      if (reconnectTimeoutRef.current) clearTimeout(reconnectTimeoutRef.current)
      if (resizeTimeout) clearTimeout(resizeTimeout)
      resizeObserver.disconnect()
      wsRef.current?.close()
      term.dispose()
      termRef.current = null
      fitRef.current = null
    }
  }, [session, pane, fontSize])

  const handleFitWindow = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN && termRef.current && fitRef.current) {
      fitRef.current.fit()
      const cols = termRef.current.cols
      const rows = termRef.current.rows
      wsRef.current.send(`3${JSON.stringify({ columns: cols, rows: rows })}`)
    }
  }, [])

  return (
    <div className="mobile-terminal-wrapper flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="mobile-terminal-area relative flex-none">
        <div
          ref={containerRef}
          className="mobile-terminal-container h-[calc((var(--app-height,100vh)-48px)/2)] flex-none min-h-0 overflow-hidden bg-term-bg"
        />
        <button
          type="button"
          className="mobile-fit-window-btn absolute bottom-2 right-2 z-50 flex items-center justify-center gap-0.5 whitespace-nowrap rounded-[14px] border-none bg-white/12 px-2 py-1 text-[11px] text-white/60 cursor-pointer tap-none"
          onClick={handleFitWindow}
          title="Fit window"
        >
          <Maximize2 size={12} />
          <span>Fit</span>
        </button>
      </div>
      <MobileToolbox
        onSend={sendText}
        disabled={false}
        fontSize={fontSize}
        onFontSizeChange={onFontSizeChange}
        voiceRef={voiceRef}
        keyboardMode={showKeyboard}
        onToggleKeyboard={toggleKeyboard}
        taskHistoryPaneKey={taskHistoryPaneKey}
        onStatusChange={onStatusChange}
      />
    </div>
  )
}
