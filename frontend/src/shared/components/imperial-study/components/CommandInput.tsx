// CommandInput.tsx — Command input for dispatching to Butler

import { Loader2, Send } from 'lucide-react'
import { useCallback, useMemo, useRef, useState } from 'react'
import { getAuthHeader } from '../../../../utils/auth'
import { VoiceInput } from '../../VoiceInput'
import { BUTLER_API_BASE } from '../constants'
import type { RoutingInfo } from '../types'

const ASSISTANT_TAGS = [
  { id: 'translator', label: 'Translate', color: '#8b5cf6' },
  { id: 'cli', label: 'CLI', color: '#06b6d4' },
  { id: 'market', label: 'Market', color: '#f59e0b' },
  { id: 'chat', label: 'Chat', color: '#ec4899' },
] as const

interface CommandInputProps {
  onDispatched?: (result: {
    run_id: string
    task_id: string
    routing?: RoutingInfo
    intent: string
  }) => void
  paneTarget?: string | null
  activeTag: string | null
  onTagChange: (tag: string | null) => void
  onAssistantSend?: (content: string, assistantType: string) => void
}

export function CommandInput({
  onDispatched,
  paneTarget,
  activeTag,
  onTagChange,
  onAssistantSend,
}: CommandInputProps) {
  const [intent, setIntent] = useState('')
  const [loading, setLoading] = useState(false)
  const [feedback, setFeedback] = useState<{ type: 'ok' | 'err'; msg: string } | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const clearFeedback = useCallback(() => {
    setFeedback(null)
  }, [])

  const resetHeight = useCallback(() => {
    const el = textareaRef.current
    if (el) el.style.height = 'auto'
  }, [])

  const resetInput = useCallback(() => {
    setIntent('')
    resetHeight()
  }, [resetHeight])

  const activeTagLabel = useMemo(() => {
    if (!activeTag) return null
    return ASSISTANT_TAGS.find((t) => t.id === activeTag)?.label ?? null
  }, [activeTag])

  const handleSubmit = useCallback(async () => {
    const trimmed = intent.trim()
    if (!trimmed || loading) return

    // If a tag is selected, route to assistant pane
    if (activeTag) {
      onAssistantSend?.(trimmed, activeTag)
      resetInput()
      return
    }

    setLoading(true)
    setFeedback(null)

    try {
      const payload: Record<string, unknown> = { intent: trimmed }
      if (paneTarget) {
        payload.params = { pane_target: paneTarget }
      }
      const authHeader = getAuthHeader()
      const res = await fetch(`${BUTLER_API_BASE}/orchestrate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(authHeader ? { Authorization: authHeader } : {}),
        },
        body: JSON.stringify(payload),
      })
      const data = await res.json()

      if (data.success) {
        // Chat fallback: auto-switch to chat mode
        if (data.data?.chat_fallback) {
          setFeedback({ type: 'ok', msg: 'Chat mode' })
          onAssistantSend?.(trimmed, 'chat')
          onTagChange('chat')
          resetInput()
          setTimeout(clearFeedback, 3000)
          return
        }

        setFeedback({ type: 'ok', msg: `Dispatched \u00b7 run ${data.data.run_id?.slice(0, 8)}` })
        resetInput()
        onDispatched?.({ ...data.data, intent: trimmed })
        // Auto-clear feedback
        setTimeout(clearFeedback, 4000)
      } else {
        const errMsg = data.error?.message || data.detail || 'Dispatch failed'
        setFeedback({ type: 'err', msg: errMsg })
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Network error'
      setFeedback({ type: 'err', msg })
    } finally {
      setLoading(false)
    }
  }, [
    intent,
    loading,
    onDispatched,
    clearFeedback,
    paneTarget,
    activeTag,
    onAssistantSend,
    onTagChange,
    resetInput,
  ])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      // Cmd/Ctrl + Enter to send
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        handleSubmit()
      }
    },
    [handleSubmit],
  )

  // Auto-resize textarea to fit content
  const autoResize = useCallback(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${Math.min(el.scrollHeight, 120)}px`
  }, [])

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setIntent(e.target.value)
      // defer resize to next tick so value is committed
      requestAnimationFrame(autoResize)
    },
    [autoResize],
  )

  return (
    <div className="is-command">
      <div className="is-command__box">
        <textarea
          ref={textareaRef}
          className="is-command__input"
          value={intent}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={
            activeTagLabel
              ? `${activeTagLabel}... (\u2318+Enter)`
              : 'Enter command... (\u2318+Enter to send)'
          }
          disabled={loading}
          rows={1}
        />
        <VoiceInput
          onText={(text) =>
            setIntent((prev) => {
              const base = prev.replace(/\s*\[.*\]\s*$/, '').trim()
              return base ? `${base} ${text}` : text
            })
          }
          onPartial={(text) =>
            setIntent((prev) => {
              const base = prev.replace(/\s*\[.*\]\s*$/, '').trim()
              return base ? `${base} [${text}]` : `[${text}]`
            })
          }
          disabled={loading}
        />
        <button
          type="button"
          className={`is-command__send ${loading ? 'loading' : ''}`}
          onClick={handleSubmit}
          disabled={!intent.trim() || loading}
          title="Send"
        >
          {loading ? <Loader2 size={16} /> : <Send size={16} />}
        </button>
      </div>
      <div className="is-command__tags">
        {ASSISTANT_TAGS.map((tag) => (
          <button
            type="button"
            key={tag.id}
            className={`is-command__tag ${activeTag === tag.id ? 'active' : ''}`}
            style={{ '--tag-color': tag.color } as React.CSSProperties}
            onClick={() => onTagChange(activeTag === tag.id ? null : tag.id)}
          >
            {tag.label}
          </button>
        ))}
      </div>
      {feedback && <div className={`is-command__feedback ${feedback.type}`}>{feedback.msg}</div>}
    </div>
  )
}
