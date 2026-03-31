import { useCallback } from 'react'
import { getAuthHeader } from '../utils/auth'

export function useAuthFetch() {
  const authFetch = useCallback(async (url: string, options: RequestInit = {}) => {
    const auth = getAuthHeader()
    const headers: Record<string, string> = {
      ...(options.headers as Record<string, string>),
    }
    if (auth) headers.Authorization = auth
    return fetch(url, { ...options, headers })
  }, [])

  return { authFetch }
}
