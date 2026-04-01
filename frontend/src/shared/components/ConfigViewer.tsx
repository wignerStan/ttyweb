import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { getAuthHeaders } from '../../utils/auth'

interface ConfigFile {
  content: Record<string, unknown> | null
  path: string
  missing?: boolean
  error?: string
}

interface ConfigData {
  opencode?: ConfigFile
  oh_my_opencode?: ConfigFile
}

type ConfigTabId = 'opencode' | 'oh_my_opencode'

const TAB_LABELS: Record<ConfigTabId, string> = {
  opencode: 'opencode.json',
  oh_my_opencode: 'oh-my-opencode.json',
}

interface ConfigViewerProps {
  paneKey?: string | null
}

export function ConfigViewer({ paneKey }: ConfigViewerProps) {
  const [data, setData] = useState<ConfigData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<ConfigTabId>('opencode')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    const params = new URLSearchParams()
    if (paneKey) params.set('paneKey', paneKey)

    const headers = getAuthHeaders()
    fetch(`/api/opencode-config?${params}`, { headers })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.json()
      })
      .then((json) => {
        if (!cancelled) {
          setData(json)
          setLoading(false)
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err.message)
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [paneKey])

  if (loading) {
    return (
      <div className="flex flex-col h-full overflow-hidden">
        <div className="flex items-center justify-center gap-2 h-full text-on-surface-muted text-xs">
          <Loader2 size={14} className="animate-spin" />
          <span>加载配置...</span>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col h-full overflow-hidden">
        <div className="p-3 text-error text-xs">加载失败: {error}</div>
      </div>
    )
  }

  const tabs: ConfigTabId[] = ['opencode', 'oh_my_opencode']
  const current = data?.[activeTab]

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="flex gap-0.5 px-1 shrink-0 bg-base-300 border-b border-base-200">
        {tabs.map((tab) => (
          <button
            type="button"
            key={tab}
            className={`flex-1 px-2 py-1.5 bg-transparent border-none border-b-2 border-transparent text-xs cursor-pointer transition-all duration-150 whitespace-nowrap overflow-hidden truncate ${
              activeTab === tab
                ? 'text-primary border-b-primary'
                : 'text-on-surface-muted hover:text-on-surface'
            }`}
            onClick={() => setActiveTab(tab)}
          >
            {TAB_LABELS[tab]}
          </button>
        ))}
      </div>

      {current?.path && (
        <div
          className="px-2 py-1 text-2xs text-on-surface-muted bg-base-300 border-b border-base-200 shrink-0 whitespace-nowrap overflow-hidden truncate"
          title={current.path}
        >
          📂 {current.path}
        </div>
      )}

      <div className="flex-1 min-h-0 overflow-auto p-2 scrollbar-thin">
        {current?.missing ? (
          <div className="flex items-center justify-center h-full text-on-surface-muted text-xs">
            文件不存在
          </div>
        ) : current?.error ? (
          <div className="p-3 text-error text-xs">{current.error}</div>
        ) : current?.content ? (
          <pre className="m-0 font-mono text-xs leading-normal text-on-surface whitespace-pre-wrap break-all">
            {JSON.stringify(current.content, null, 2)}
          </pre>
        ) : (
          <div className="flex items-center justify-center h-full text-on-surface-muted text-xs">
            无数据
          </div>
        )}
      </div>
    </div>
  )
}
