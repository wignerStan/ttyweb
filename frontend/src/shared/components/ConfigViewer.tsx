import { Loader2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { getAuthHeader } from '../../utils/auth'

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

    const auth = getAuthHeader()
    const headers: Record<string, string> = {}
    if (auth) headers.Authorization = auth
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
        <div className="flex items-center justify-center gap-2 h-full text-[var(--zinc-500)] text-xs">
          <Loader2 size={14} className="animate-spin" />
          <span>加载配置...</span>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col h-full overflow-hidden">
        <div className="p-3 text-[var(--red-400)] text-xs">加载失败: {error}</div>
      </div>
    )
  }

  const tabs: ConfigTabId[] = ['opencode', 'oh_my_opencode']
  const current = data?.[activeTab]

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="flex gap-0.5 px-1 shrink-0 bg-[var(--zinc-900)] border-b border-[var(--zinc-800)]">
        {tabs.map((tab) => (
          <button
            type="button"
            key={tab}
            className={`flex-1 px-2 py-1.5 bg-transparent border-none border-b-2 border-transparent text-[11px] cursor-pointer transition-all duration-150 whitespace-nowrap overflow-hidden truncate ${
              activeTab === tab
                ? 'text-[var(--blue-500)] border-b-[var(--blue-500)]'
                : 'text-[var(--zinc-500)] hover:text-[var(--zinc-300)]'
            }`}
            onClick={() => setActiveTab(tab)}
          >
            {TAB_LABELS[tab]}
          </button>
        ))}
      </div>

      {current?.path && (
        <div
          className="px-2 py-1 text-[10px] text-[var(--zinc-600)] bg-[var(--zinc-900)] border-b border-[var(--zinc-800)] shrink-0 whitespace-nowrap overflow-hidden truncate"
          title={current.path}
        >
          📂 {current.path}
        </div>
      )}

      <div className="flex-1 min-h-0 overflow-auto p-2 scrollbar-thin">
        {current?.missing ? (
          <div className="flex items-center justify-center h-full text-[var(--zinc-600)] text-xs">
            文件不存在
          </div>
        ) : current?.error ? (
          <div className="p-3 text-[var(--red-400)] text-xs">{current.error}</div>
        ) : current?.content ? (
          <pre className="m-0 font-mono text-[11px] leading-normal text-[var(--zinc-300)] whitespace-pre-wrap break-all">
            {JSON.stringify(current.content, null, 2)}
          </pre>
        ) : (
          <div className="flex items-center justify-center h-full text-[var(--zinc-600)] text-xs">
            无数据
          </div>
        )}
      </div>
    </div>
  )
}
