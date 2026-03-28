import { useState, useCallback, useEffect, useRef } from 'react'
import { Send, Copy, Play, Loader2, ChevronUp, ChevronDown, ChevronRight, X, Terminal, Square } from 'lucide-react'
import { RoleManagerModal } from './RoleManagerModal'
import { getAuthHeader } from '../../utils/auth'
import type { AiRole } from '../../types'

const TEMPLATE_ROLE_ID = 'research-publish'

const BUILTIN_ROLES: AiRole[] = [
  { id: 'cli', emoji: '\u{1F5A5}\u{FE0F}', label: '\u547D\u4EE4\u884C\u5927\u795E', desc: '\u751F\u6210\u53EF\u6267\u884C\u7684\u7EC8\u7AEF\u547D\u4EE4' },
  { id: 'ops', emoji: '\u{1F527}', label: '\u8FD0\u7EF4\u4E13\u5BB6', desc: '\u4F18\u5316 DevOps/\u8FD0\u7EF4\u63D0\u793A\u8BCD' },
  { id: 'prompt', emoji: '\u{2728}', label: '\u63D0\u793A\u8BCD\u4F18\u5316', desc: '\u901A\u7528 AI \u63D0\u793A\u8BCD\u4F18\u5316' },
  { id: 'frontend', emoji: '\u{1F3A8}', label: '\u524D\u7AEF\u4F18\u5316', desc: '\u524D\u7AEF\u5F00\u53D1\u63D0\u793A\u8BCD\u4F18\u5316' },
  { id: 'backend', emoji: '\u{2699}\u{FE0F}', label: '\u540E\u7AEF\u4F18\u5316', desc: '\u540E\u7AEF\u5F00\u53D1\u63D0\u793A\u8BCD\u4F18\u5316' },
  { id: 'ui', emoji: '\u{1F3AD}', label: 'UI\u4F18\u5316', desc: 'UI/UX \u8BBE\u8BA1\u63D0\u793A\u8BCD\u4F18\u5316' },
  { id: 'api', emoji: '\u{1F504}', label: 'API\u8F6C\u6362', desc: 'API \u67B6\u6784\u8F6C\u6362\u4E0E\u91CD\u6784' },
]

interface AiCommandTabProps {
  onSend: (text: string) => void
  disabled?: boolean
  initialText?: string
  onTextConsumed?: () => void
}

const PRE_STYLE: React.CSSProperties = {
  background: '#13151a',
  borderRadius: '4px',
  padding: '8px',
  color: '#98c379',
  fontSize: '12px',
  fontFamily: 'Menlo, Monaco, monospace',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  margin: 0,
  overflow: 'auto',
}

function CopyExecuteButtons({ onCopy, onExecute, disabled }: {
  onCopy: () => void
  onExecute: () => void
  disabled?: boolean
}) {
  return (
    <div style={{ display: 'flex', gap: '6px', marginTop: '6px' }}>
      <button
        onClick={onCopy}
        style={{
          flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
          gap: '4px', padding: '6px', background: '#2c313a', color: '#abb2bf',
          border: 'none', borderRadius: '4px', fontSize: '12px', cursor: 'pointer',
        }}
        type="button"
      >
        <Copy size={12} /> \u590D\u5236
      </button>
      <button
        onClick={onExecute}
        disabled={disabled}
        style={{
          flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
          gap: '4px', padding: '6px', background: '#4d78cc', color: '#fff',
          border: 'none', borderRadius: '4px', fontSize: '12px', cursor: 'pointer',
          opacity: disabled ? 0.5 : 1,
        }}
        type="button"
      >
        <Play size={12} /> \u6267\u884C
      </button>
    </div>
  )
}

export function AiCommandTab({ onSend, disabled, initialText, onTextConsumed }: AiCommandTabProps) {
  const [input, setInput] = useState('')
  const [selectedRole, setSelectedRole] = useState('cli')
  const [roles, setRoles] = useState<AiRole[]>(BUILTIN_ROLES)
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{ command: string; explanation: string } | null>(null)
  const [expanded, setExpanded] = useState(false)
  const [showPrompt, setShowPrompt] = useState(false)
  const [showRoleModal, setShowRoleModal] = useState(false)
  const [showRoleDropdown, setShowRoleDropdown] = useState(false)
  const [streaming, setStreaming] = useState(false)
  const [streamText, setStreamText] = useState('')
  const wsRef = useRef<WebSocket | null>(null)
  const streamTextRef = useRef('')
  const resultRef = useRef<HTMLDivElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (initialText) {
      setInput(initialText)
      onTextConsumed?.()
    }
  }, [initialText, onTextConsumed])

  // Auto-scroll streaming result
  useEffect(() => {
    if (resultRef.current) {
      resultRef.current.scrollTop = resultRef.current.scrollHeight
    }
  }, [streamText])

  // Close dropdown on outside click
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowRoleDropdown(false)
      }
    }
    if (showRoleDropdown) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showRoleDropdown])

  const fetchRoles = useCallback(async () => {
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = {}
      if (auth) headers['Authorization'] = auth
      const res = await fetch('/api/roles', { headers })
      if (res.ok) {
        const data = await res.json()
        if (data.roles?.length) setRoles(data.roles)
      }
    } catch {
      // fallback to built-in
    }
  }, [])

  useEffect(() => { fetchRoles() }, [fetchRoles])

  const resetStream = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setStreaming(false)
    setStreamText('')
    streamTextRef.current = ''
  }, [])

  const stopStreaming = useCallback(() => {
    resetStream()
    if (streamTextRef.current) {
      setResult({ command: streamTextRef.current, explanation: '' })
      streamTextRef.current = ''
    }
  }, [resetStream])

  const handleDirectSend = useCallback(() => {
    if (!input.trim() || disabled) return
    onSend(input.trim() + '\n')
    setInput('')
  }, [input, onSend, disabled])

  const handleClear = useCallback(() => {
    setInput('')
    setResult(null)
    resetStream()
    setLoading(false)
  }, [resetStream])

  const fallbackNonStreaming = useCallback(async (prompt: string) => {
    setLoading(true)
    try {
      const auth = getAuthHeader()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (auth) headers['Authorization'] = auth
      const res = await fetch('/api/ai/command', {
        method: 'POST',
        headers,
        body: JSON.stringify({ prompt, role: selectedRole })
      })
      const data = await res.json()
      setResult({ command: data.command || '', explanation: data.explanation || '' })
    } catch (err) {
      setResult({ command: '', explanation: '\u8BF7\u6C42\u5931\u8D25: ' + (err instanceof Error ? err.message : String(err)) })
    } finally {
      setLoading(false)
    }
  }, [selectedRole])

  const handleGenerate = useCallback(async () => {
    if (!input.trim() || loading || streaming) return
    if (selectedRole === TEMPLATE_ROLE_ID) {
      const url = input.trim()
      setInput(`\u7528 github-project-researcher \u7814\u7A76 ${url}\uFF0C\u7136\u540E\u7528 md2wechat \u53D1\u5FAE\u4FE1\u516C\u4F17\u53F7`)
      return
    }

    setLoading(true)
    setResult(null)
    setStreamText('')
    streamTextRef.current = ''

    const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}${location.pathname}ws/ai/stream`

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws
      setStreaming(true)
      setLoading(false)

      ws.onopen = () => {
        ws.send(JSON.stringify({ role: selectedRole, prompt: input.trim() }))
      }

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === 'token') {
            streamTextRef.current += msg.data
            setStreamText(streamTextRef.current)
          } else if (msg.type === 'done') {
            setResult({ command: msg.data || streamTextRef.current, explanation: '' })
            resetStream()
          } else if (msg.type === 'error') {
            setResult({ command: '', explanation: '\u9519\u8BEF: ' + msg.data })
            resetStream()
          }
        } catch {
          // ignore parse errors
        }
      }

      ws.onerror = () => {
        resetStream()
        fallbackNonStreaming(input.trim())
      }

      ws.onclose = () => {
        if (wsRef.current === ws) {
          wsRef.current = null
        }
      }
    } catch {
      fallbackNonStreaming(input.trim())
    }
  }, [input, selectedRole, loading, streaming, resetStream, fallbackNonStreaming])

  const handleCopy = useCallback(async () => {
    const text = streaming ? streamText : result?.command
    if (text) {
      try {
        await navigator.clipboard.writeText(text)
      } catch { /* ignore */ }
    }
  }, [result, streaming, streamText])

  const handleExecute = useCallback(() => {
    const text = streaming ? streamText : result?.command
    if (text) {
      onSend(text + '\n')
    }
  }, [result, onSend, streaming, streamText])

  const selectedRoleDef = roles.find(r => r.id === selectedRole)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: '8px', gap: '8px', overflow: 'auto' }}>
      {/* Role selector dropdown */}
      <div style={{ position: 'relative' }} ref={dropdownRef}>
        <button
          onClick={() => setShowRoleDropdown(!showRoleDropdown)}
          style={{
            width: '100%',
            padding: '6px 10px',
            borderRadius: '8px',
            border: '1px solid #2c313a',
            background: '#1a1c20',
            color: '#abb2bf',
            fontSize: '13px',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '6px',
            textAlign: 'left',
          }}
          type="button"
        >
          <span style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <span>{selectedRoleDef?.emoji ?? '\u26A1'}</span>
            <span style={{ fontWeight: 500 }}>{selectedRoleDef?.label ?? '\u547D\u4EE4\u884C\u5927\u795E'}</span>
          </span>
          <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
            <span style={{ color: '#555a66', fontSize: '11px' }}>{selectedRoleDef?.desc}</span>
            <ChevronRight
              size={14}
              style={{
                color: '#7a818c',
                transform: showRoleDropdown ? 'rotate(90deg)' : 'none',
                transition: 'transform 0.15s',
                flexShrink: 0,
              }}
            />
          </span>
        </button>

        {showRoleDropdown && (
          <div style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            marginTop: '4px',
            background: '#1e2028',
            border: '1px solid #2c313a',
            borderRadius: '8px',
            zIndex: 100,
            maxHeight: '240px',
            overflowY: 'auto',
            boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
          }}>
            {roles.filter(r => r.id !== TEMPLATE_ROLE_ID).map(role => (
              <button
                key={role.id}
                onClick={() => {
                  setSelectedRole(role.id)
                  setShowRoleDropdown(false)
                  setShowPrompt(false)
                }}
                style={{
                  width: '100%',
                  padding: '8px 10px',
                  background: role.id === selectedRole ? '#4d78cc22' : 'none',
                  border: 'none',
                  borderBottom: '1px solid #2c313a',
                  color: role.id === selectedRole ? '#4d78cc' : '#abb2bf',
                  fontSize: '13px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  textAlign: 'left',
                }}
                type="button"
              >
                <span>{role.emoji}</span>
                <span style={{ flex: 1 }}>
                  <span style={{ fontWeight: 500 }}>{role.label}</span>
                  <span style={{ color: '#555a66', fontSize: '11px', marginLeft: '6px' }}>{role.desc}</span>
                </span>
              </button>
            ))}
            <button
              onClick={() => {
                setShowRoleDropdown(false)
                setShowRoleModal(true)
              }}
              style={{
                width: '100%',
                padding: '8px 10px',
                background: 'none',
                border: 'none',
                color: '#7a818c',
                fontSize: '12px',
                cursor: 'pointer',
                textAlign: 'center',
              }}
              type="button"
            >
              + \u7BA1\u7406\u89D2\u8272
            </button>
          </div>
        )}
      </div>

      {/* Prompt viewer toggle */}
      <button
        onClick={() => setShowPrompt(!showPrompt)}
        style={{
          background: 'none',
          border: 'none',
          color: '#555a66',
          fontSize: '11px',
          cursor: 'pointer',
          padding: '0',
          display: 'flex',
          alignItems: 'center',
          gap: '2px',
        }}
        type="button"
      >
        {showPrompt ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
        \u7CFB\u7EDF\u63D0\u793A\u8BCD
      </button>

      {/* Prompt viewer content */}
      {showPrompt && selectedRoleDef?.prompt && (
        <div style={{
          background: '#1a1c20',
          border: '1px solid #2c313a',
          borderRadius: '6px',
          padding: '8px',
          fontSize: '11px',
          color: '#7a818c',
          maxHeight: '120px',
          overflow: 'auto',
          whiteSpace: 'pre-wrap',
        }}>
          <div style={{ color: '#9da5b4', marginBottom: '4px', fontWeight: 600 }}>{selectedRoleDef.emoji} {selectedRoleDef.label}</div>
          {selectedRoleDef.prompt}
          {selectedRoleDef.suffix && (
            <div style={{ marginTop: '8px', color: '#555a66', borderTop: '1px solid #2c313a', paddingTop: '4px' }}>
              {selectedRoleDef.suffix}
            </div>
          )}
        </div>
      )}

      {/* Input area */}
      <div style={{ position: 'relative' }}>
        <textarea
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              if (streaming) return
              handleGenerate()
            }
          }}
          placeholder="\u63CF\u8FF0\u4F60\u60F3\u6267\u884C\u7684\u64CD\u4F5C..."
          rows={2}
          disabled={streaming}
          style={{
            width: '100%',
            background: '#13151a',
            border: '1px solid #2c313a',
            borderRadius: '6px',
            padding: '8px',
            paddingRight: '28px',
            color: '#abb2bf',
            fontSize: '13px',
            outline: 'none',
            resize: 'none',
            fontFamily: 'inherit',
            boxSizing: 'border-box',
            opacity: streaming ? 0.6 : 1,
          }}
        />
        {input && !streaming && (
          <button
            onClick={handleClear}
            style={{
              position: 'absolute',
              top: '6px',
              right: '6px',
              background: 'none',
              border: 'none',
              color: '#555a66',
              cursor: 'pointer',
              padding: '2px',
              borderRadius: '4px',
              display: 'flex',
              alignItems: 'center',
            }}
            type="button"
            title="\u6E05\u7A7A"
          >
            <X size={14} />
          </button>
        )}
      </div>

      {/* Action buttons */}
      <div style={{ display: 'flex', gap: '6px' }}>
        {streaming ? (
          <button
            onClick={stopStreaming}
            style={{
              flex: 1,
              padding: '8px 12px',
              background: '#e06c75',
              color: '#fff',
              border: 'none',
              borderRadius: '6px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '4px',
              fontSize: '12px',
              whiteSpace: 'nowrap',
            }}
            type="button"
          >
            <Square size={14} /> \u505C\u6B62
          </button>
        ) : (
          <button
            onClick={handleGenerate}
            disabled={!input.trim() || loading || disabled}
            style={{
              flex: 1,
              padding: '8px 12px',
              background: loading ? '#2c313a' : '#4d78cc',
              color: '#fff',
              border: 'none',
              borderRadius: '6px',
              cursor: loading ? 'default' : 'pointer',
              opacity: (!input.trim() || disabled) ? 0.5 : 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '4px',
              fontSize: '12px',
              whiteSpace: 'nowrap',
            }}
            type="button"
          >
            {loading ? <Loader2 size={14} className="spin" /> : <Send size={14} />}
            AI \u751F\u6210
          </button>
        )}
        <button
          onClick={handleClear}
          disabled={!input && !result && !streamText}
          style={{
            padding: '8px 10px',
            background: '#2c313a',
            color: '#9da5b4',
            border: 'none',
            borderRadius: '6px',
            cursor: 'pointer',
            opacity: (!input && !result && !streamText) ? 0.3 : 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '12px',
          }}
          type="button"
          title="\u6E05\u7A7A"
        >
          <X size={14} />
        </button>
        <button
          onClick={handleDirectSend}
          disabled={!input.trim() || disabled || streaming}
          style={{
            padding: '8px 12px',
            background: '#2c313a',
            color: '#9da5b4',
            border: 'none',
            borderRadius: '6px',
            cursor: 'pointer',
            opacity: (!input.trim() || disabled || streaming) ? 0.3 : 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '4px',
            fontSize: '12px',
            whiteSpace: 'nowrap',
          }}
          type="button"
          title="\u76F4\u63A5\u53D1\u9001\u5230\u7EC8\u7AEF"
        >
          <Terminal size={14} /> \u53D1\u9001\u7EC8\u7AEF
        </button>
      </div>

      {/* Streaming output */}
      {streaming && streamText && (
        <div ref={resultRef} style={{
          background: '#1a1c20',
          border: '1px solid #4d78cc44',
          borderRadius: '6px',
          padding: '8px',
        }}>
          <pre style={{ ...PRE_STYLE, maxHeight: '200px' }}>
            {streamText}
            <span style={{ color: '#4d78cc' }}>&#9646;</span>
          </pre>
          <CopyExecuteButtons onCopy={handleCopy} onExecute={handleExecute} disabled={disabled} />
        </div>
      )}

      {/* Result card (non-streaming) */}
      {result && !streaming && (
        <div style={{
          background: '#1a1c20',
          border: '1px solid #2c313a',
          borderRadius: '6px',
          padding: '8px',
          display: 'flex',
          flexDirection: 'column',
          gap: '6px',
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ color: '#7a818c', fontSize: '11px' }}>{result.explanation}</span>
            <div style={{ display: 'flex', gap: '4px' }}>
              <button
                onClick={() => setExpanded(!expanded)}
                style={{ background: 'none', border: 'none', color: '#7a818c', cursor: 'pointer', padding: '2px' }}
                type="button"
              >
                {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
              </button>
            </div>
          </div>
          {result.command && (
            <>
              <pre style={{ ...PRE_STYLE, maxHeight: expanded ? 'none' : '80px', overflow: expanded ? 'auto' : 'hidden' }}>
                {result.command}
              </pre>
              <CopyExecuteButtons onCopy={handleCopy} onExecute={handleExecute} disabled={disabled} />
            </>
          )}
        </div>
      )}

      <RoleManagerModal
        open={showRoleModal}
        onClose={() => setShowRoleModal(false)}
        roles={roles}
        onRolesChanged={fetchRoles}
      />
    </div>
  )
}
