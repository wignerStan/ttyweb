const AUTH_KEY = 'ttyweb_auth'

export async function login(
  username: string,
  password: string,
): Promise<{ success: boolean; error?: string }> {
  const credentials = btoa(`${username}:${password}`)
  try {
    const res = await fetch('/api/auth/check', {
      headers: { Authorization: `Basic ${credentials}` },
    })
    if (!res.ok) {
      const data = await res.json()
      return { success: false, error: data.error || 'Invalid credentials' }
    }
    localStorage.setItem(AUTH_KEY, credentials)
    return { success: true }
  } catch {
    return { success: false, error: 'Connection failed' }
  }
}

export async function logout(): Promise<void> {
  localStorage.removeItem(AUTH_KEY)
}

export async function checkAuth(): Promise<boolean> {
  const credentials = localStorage.getItem(AUTH_KEY)
  try {
    const headers: Record<string, string> = {}
    if (credentials) headers.Authorization = `Basic ${credentials}`
    const res = await fetch('/api/auth/check', { headers })
    if (!res.ok) return false
    const data = await res.json()
    // If auth is not enabled, consider the user authenticated
    if (data.data?.auth_enabled === false) return true
    // If auth is enabled, credentials must be present
    return !!credentials
  } catch {
    return false
  }
}

export function getAuthHeader(): string | null {
  const credentials = localStorage.getItem(AUTH_KEY)
  return credentials ? `Basic ${credentials}` : null
}

export function getAuthHeaders(): Record<string, string> {
  const auth = getAuthHeader()
  return auth ? { Authorization: auth } : {}
}

export function getToken(): string {
  return localStorage.getItem(AUTH_KEY) || ''
}
