import { useCallback } from 'react'
import { getAuthHeader } from '../utils/auth'

function mergeHeaders(input: RequestInit['headers'], auth: string | null): HeadersInit {
  const result = new Headers(input)
  if (auth) {
    result.set('Authorization', auth)
  }
  return result
}

/**
 * React hook for authenticated fetch requests.
 * Automatically attaches Authorization header from stored credentials.
 */
export function useAuthFetch() {
  const authFetch = useCallback(async (url: string, options: RequestInit = {}) => {
    const auth = getAuthHeader()
    return fetch(url, { ...options, headers: mergeHeaders(options.headers, auth) })
  }, [])

  return { authFetch }
}
