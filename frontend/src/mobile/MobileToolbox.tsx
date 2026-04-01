import { Bot, Clock, Grid3X3, Settings, Upload } from 'lucide-react'
import { type RefObject, useCallback, useRef, useState } from 'react'
import { useTmuxPrefix } from '../hooks/useTmuxPrefix'
import { AiCommandTab } from '../shared/components/AiCommandTab'
import { ConfigViewer } from '../shared/components/ConfigViewer'
import { FileUpload } from '../shared/components/FileUpload'
import { SnippetsTab } from '../shared/components/SnippetsTab'
import { VoiceInput, type VoiceInputHandle } from '../shared/components/VoiceInput'

const preventFocus = (e: React.MouseEvent | React.TouchEvent) => {
  e.preventDefault()
}

interface KeyDef {
  label: string
  data?: string
  modifier?: string
}

function createKeyRows(prefixCode: string): [KeyDef[], KeyDef[]] {
  const row1: KeyDef[] = [
    { label: 'esc', data: '\x1b' },
    { label: 'tab', data: '	' },
    { label: '|', data: '|' },
    { label: '/', data: '/' },
    { label: '-', data: '-' },
    { label: '~', data: '~' },
    { label: '^C', data: '\x03' },
    { label: 'clr', data: '\x15' },
  ]

  const row2: KeyDef[] = [
    { label: 'ctrl', modifier: 'ctrl' },
    { label: 'alt', modifier: 'alt' },
    { label: '\u2191', data: '\x1b[A' },
    { label: '\u2193', data: '\x1b[B' },
    { label: '\u2190', data: '\x1b[D' },
    { label: '\u2192', data: '\x1b[C' },
    { label: '\ud83d\udcdc', data: `${prefixCode}[` },
    { label: '\u23ce', data: '\r' },
  ]

  return [row1, row2]
}

type TabId = 'snippets' | 'ai' | 'upload' | 'config'

interface MobileToolboxProps {
  onSend: (text: string) => void
  disabled?: boolean
  fontSize: number
  onFontSizeChange: (size: number) => void
  voiceRef?: RefObject<VoiceInputHandle | null>
  keyboardMode?: boolean
  onToggleKeyboard?: () => void
  taskHistoryPaneKey?: string | null
  onStatusChange?: () => void
}

export function MobileToolbox({
  onSend,
  disabled,
  fontSize,
  onFontSizeChange,
  voiceRef,
  keyboardMode,
  onToggleKeyboard,
  taskHistoryPaneKey: _taskHistoryPaneKey,
  onStatusChange: _onStatusChange,
}: MobileToolboxProps) {
  const [activeTab, setActiveTab] = useState<TabId>('ai')
  const [ctrlActive, setCtrlActive] = useState(false)
  const [altActive, setAltActive] = useState(false)
  const [voiceText, setVoiceText] = useState<string | undefined>(undefined)
  const localVoiceRef = useRef<VoiceInputHandle | null>(null)
  const effectiveVoiceRef = voiceRef || localVoiceRef

  const prefix = useTmuxPrefix()
  const [keyRow1, keyRow2] = createKeyRows(prefix.code)

  const handleKey = useCallback(
    (key: KeyDef) => {
      if (key.modifier === 'ctrl') {
        setCtrlActive((prev) => !prev)
        return
      }
      if (key.modifier === 'alt') {
        setAltActive((prev) => !prev)
        return
      }
      if (!key.data) return

      let data = key.data
      if (ctrlActive && data.length === 1) {
        const upper = data.toUpperCase()
        if (upper >= 'A' && upper <= 'Z') {
          data = String.fromCharCode(upper.charCodeAt(0) - 64)
        }
        setCtrlActive(false)
      }
      if (altActive && data.length === 1) {
        data = `\x1b${data}`
        setAltActive(false)
      }
      onSend(data)
    },
    [ctrlActive, altActive, onSend],
  )

  const handleVoiceText = useCallback((text: string) => {
    setVoiceText(text)
    setActiveTab('ai')
  }, [])

  const handleTextConsumed = useCallback(() => {
    setVoiceText(undefined)
  }, [])

  if (keyboardMode) {
    return (
      <div className="mobile-toolbox keyboard-mode flex flex-col overflow-hidden border-t border-base-200 bg-base-100 safe-area-b">
        <div className="toolbox-keys shrink-0 border-b border-base-200">
          <div className="toolbox-key-row flex min-h-[44px]">
            {keyRow1.map((k) => (
              <button
                key={k.label}
                className="toolbox-key btn btn-ghost min-h-[44px] min-w-[44px] flex-1 border-b border-r border-base-200 bg-base-300 px-0 py-0 font-mono text-xs text-base-content/75 cursor-pointer tap-none select-none"
                onMouseDown={preventFocus}
                onTouchStart={preventFocus}
                onTouchEnd={(e) => {
                  e.preventDefault()
                  handleKey(k)
                }}
                type="button"
              >
                {k.label}
              </button>
            ))}
          </div>
          <div className="toolbox-key-row flex min-h-[44px]">
            {keyRow2.map((k) => (
              <button
                key={k.label}
                className={`toolbox-key btn btn-ghost min-h-[44px] min-w-[44px] flex-1 border-b border-r border-base-200 bg-base-300 px-0 py-0 font-mono text-xs text-base-content/75 cursor-pointer tap-none select-none ${k.modifier === 'ctrl' && ctrlActive ? 'active bg-primary text-primary-content' : ''} ${k.modifier === 'alt' && altActive ? 'active bg-primary text-primary-content' : ''}`}
                onMouseDown={preventFocus}
                onTouchStart={preventFocus}
                onTouchEnd={(e) => {
                  e.preventDefault()
                  handleKey(k)
                }}
                type="button"
              >
                {k.label}
              </button>
            ))}
          </div>
        </div>
        <div className="keyboard-mode-bar flex h-11 items-center gap-2 border-t border-base-200 bg-base-300 safe-area-b">
          <button
            className="toolbox-grid-btn btn btn-ghost btn-circle min-h-[44px] min-w-[44px] border-r border-base-200 text-base-content/45 cursor-pointer tap-none"
            onMouseDown={preventFocus}
            onTouchStart={preventFocus}
            onTouchEnd={(e) => {
              e.preventDefault()
              onToggleKeyboard?.()
            }}
            onClick={onToggleKeyboard}
            type="button"
          >
            <Grid3X3 size={18} />
          </button>
          <span className="keyboard-mode-label text-sm text-base-content/45">收起键盘</span>
        </div>
      </div>
    )
  }

  return (
    <div className="mobile-toolbox flex h-[calc((var(--app-height,100vh)-48px)/2)] flex-none flex-col overflow-hidden border-t border-base-200 bg-base-100 safe-area-b">
      {/* Quick keys */}
      <div className="toolbox-keys shrink-0 border-b border-base-200">
        <div className="toolbox-key-row flex min-h-[44px]">
          {keyRow1.map((k) => (
            <button
              key={k.label}
              className="toolbox-key btn btn-ghost min-h-[44px] min-w-[44px] flex-1 border-b border-r border-base-200 bg-base-300 px-0 py-0 font-mono text-xs text-base-content/75 cursor-pointer tap-none select-none"
              onMouseDown={preventFocus}
              onTouchStart={preventFocus}
              onTouchEnd={(e) => {
                e.preventDefault()
                handleKey(k)
              }}
              type="button"
            >
              {k.label}
            </button>
          ))}
        </div>
        <div className="toolbox-key-row toolbox-font-row flex h-8 border-b border-base-200 bg-base-300">
          <div className="toolbox-font-slider flex w-full items-center gap-1.5 px-2.5">
            <span className="font-slider-label shrink-0 font-mono text-[11px] text-base-content/45">
              A
            </span>
            <input
              type="range"
              min="6"
              max="12"
              step="0.5"
              value={fontSize}
              onChange={(e) => onFontSizeChange(parseFloat(e.target.value))}
              onMouseDown={preventFocus}
              className="font-slider-input h-1 flex-1 appearance-none rounded bg-base-200 outline-none"
            />
            <span className="font-slider-value shrink-0 min-w-6 text-right font-mono text-[11px] text-base-content/75">
              {fontSize}
            </span>
          </div>
        </div>
        <div className="toolbox-key-row flex min-h-[44px]">
          {keyRow2.map((k) => (
            <button
              key={k.label}
              className={`toolbox-key btn btn-ghost min-h-[44px] min-w-[44px] flex-1 border-b border-r border-base-200 bg-base-300 px-0 py-0 font-mono text-xs text-base-content/75 cursor-pointer tap-none select-none ${k.modifier === 'ctrl' && ctrlActive ? 'active bg-primary text-primary-content' : ''} ${k.modifier === 'alt' && altActive ? 'active bg-primary text-primary-content' : ''}`}
              onMouseDown={preventFocus}
              onTouchStart={preventFocus}
              onTouchEnd={(e) => {
                e.preventDefault()
                handleKey(k)
              }}
              type="button"
            >
              {k.label}
            </button>
          ))}
        </div>
      </div>

      {/* Tab content */}
      <div className="toolbox-content flex-1 min-h-0 overflow-hidden">
        {activeTab === 'snippets' && <SnippetsTab onSend={onSend} disabled={disabled} />}
        {activeTab === 'ai' && (
          <AiCommandTab
            onSend={onSend}
            disabled={disabled}
            initialText={voiceText}
            onTextConsumed={handleTextConsumed}
          />
        )}
        {activeTab === 'config' && <ConfigViewer />}
        {activeTab === 'upload' && <FileUpload compact onSend={onSend} />}
      </div>

      {/* Tab bar */}
      <div className="toolbox-tabbar flex h-11 shrink-0 items-center border-t border-base-200 bg-base-300 safe-area-b">
        <button
          className="toolbox-grid-btn btn btn-ghost btn-circle min-h-[44px] min-w-[44px] border-r border-base-200 text-base-content/45 cursor-pointer tap-none"
          onMouseDown={preventFocus}
          onTouchStart={preventFocus}
          onTouchEnd={(e) => {
            e.preventDefault()
            onToggleKeyboard?.()
          }}
          onClick={onToggleKeyboard}
          type="button"
        >
          <Grid3X3 size={18} />
        </button>
        <button
          className={`toolbox-tab btn btn-ghost min-h-[44px] flex-1 items-center justify-center gap-1 text-[13px] cursor-pointer tap-none select-none ${activeTab === 'snippets' ? 'active font-semibold text-primary' : 'text-base-content/45'}`}
          onMouseDown={preventFocus}
          onTouchStart={preventFocus}
          onTouchEnd={(e) => {
            e.preventDefault()
            setActiveTab('snippets')
          }}
          type="button"
        >
          <Clock size={14} />
          <span>命令</span>
        </button>
        {/* biome-ignore lint/a11y/noStaticElementInteractions: wrapper for VoiceInput component */}
        <div
          className="toolbox-tab-voice flex h-11 min-h-[44px] w-[52px] shrink-0 items-center justify-center"
          onMouseDown={preventFocus}
          onTouchStart={preventFocus}
        >
          <VoiceInput ref={effectiveVoiceRef} onText={handleVoiceText} disabled={disabled} />
        </div>
        <button
          className={`toolbox-tab btn btn-ghost min-h-[44px] flex-1 items-center justify-center gap-1 text-[13px] cursor-pointer tap-none select-none ${activeTab === 'config' ? 'active font-semibold text-primary' : 'text-base-content/45'}`}
          onMouseDown={preventFocus}
          onTouchStart={preventFocus}
          onTouchEnd={(e) => {
            e.preventDefault()
            setActiveTab('config')
          }}
          type="button"
        >
          <Settings size={14} />
          <span>配置</span>
        </button>
        <button
          className={`toolbox-tab btn btn-ghost min-h-[44px] flex-1 items-center justify-center gap-1 text-[13px] cursor-pointer tap-none select-none ${activeTab === 'upload' ? 'active font-semibold text-primary' : 'text-base-content/45'}`}
          onMouseDown={preventFocus}
          onTouchStart={preventFocus}
          onTouchEnd={(e) => {
            e.preventDefault()
            setActiveTab('upload')
          }}
          type="button"
        >
          <Upload size={14} />
          <span>上传</span>
        </button>
        <button
          className={`toolbox-tab btn btn-ghost min-h-[44px] flex-1 items-center justify-center gap-1 text-[13px] cursor-pointer tap-none select-none ${activeTab === 'ai' ? 'active font-semibold text-primary' : 'text-base-content/45'}`}
          onMouseDown={preventFocus}
          onTouchStart={preventFocus}
          onTouchEnd={(e) => {
            e.preventDefault()
            setActiveTab('ai')
          }}
          type="button"
        >
          <Bot size={14} />
          <span>AI</span>
        </button>
      </div>
    </div>
  )
}
