import { useState, useEffect, useCallback } from 'react'
import { getAuthHeader } from '../utils/auth'

export interface QuickDir {
    name: string
    path: string
}

export function useNewWindow(onSuccess?: () => void) {
    const [quickDirs, setQuickDirs] = useState<QuickDir[]>([])
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        const auth = getAuthHeader()
        const headers: Record<string, string> = {}
        if (auth) headers['Authorization'] = auth
        fetch('/api/tmux/quick-dirs', { headers })
            .then(r => r.json())
            .then(data => setQuickDirs(data.dirs || []))
            .catch(() => setQuickDirs([]))
    }, [])

    const createWindow = useCallback(async (session: string, dir?: string, name?: string) => {
        setLoading(true)
        setError(null)
        try {
            const auth = getAuthHeader()
            const headers: Record<string, string> = { 'Content-Type': 'application/json' }
            if (auth) headers['Authorization'] = auth
            const res = await fetch('/api/tmux/new-window', {
                method: 'POST',
                headers,
                body: JSON.stringify({ session, dir, name }),
            })
            const data = await res.json()
            if (!res.ok) throw new Error(data.error || 'Failed to create window')
            onSuccess?.()
            return data
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : 'Unknown error'
            setError(msg)
            throw err
        } finally {
            setLoading(false)
        }
    }, [onSuccess])

    return { quickDirs, createWindow, loading, error }
}
