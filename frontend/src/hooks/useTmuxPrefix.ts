import { useState, useEffect } from 'react'
import { getAuthHeader } from '../utils/auth'

interface TmuxPrefix {
  code: string
  label: string
}

export function useTmuxPrefix(): TmuxPrefix {
  const [prefix, setPrefix] = useState<TmuxPrefix>({ code: '\x02', label: 'Ctrl+B' })

  useEffect(() => {
    const authHeader = getAuthHeader()
    fetch('/api/tmux/config', { headers: authHeader ? { 'Authorization': authHeader } : undefined })
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (data?.code) setPrefix({ code: data.code, label: data.label || 'prefix' })
      })
      .catch(() => {})
  }, [])

  return prefix
}
