import type { ApiResponse } from '../types'
import { getAuthHeader } from './auth'

function authHeaders(): Record<string, string> {
  const auth = getAuthHeader()
  return auth ? { Authorization: auth } : {}
}

/**
 * GET request that unwraps the ApiResponse envelope.
 * Returns `data` on success, `null` on failure.
 */
export async function apiGet<T>(url: string): Promise<T | null> {
  try {
    const res = await fetch(url, { headers: authHeaders() })
    const json: ApiResponse<T> = await res.json()
    if (json.success) return json.data
    return null
  } catch {
    return null
  }
}

/**
 * POST request with JSON body. Returns `data` on success, `null` on failure.
 */
export async function apiPost<T>(url: string, body: unknown): Promise<T | null> {
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    const json: ApiResponse<T> = await res.json()
    if (json.success) return json.data
    return null
  } catch {
    return null
  }
}

/**
 * PUT request with JSON body. Returns `data` on success, `null` on failure.
 */
export async function apiPut<T>(url: string, body: unknown): Promise<T | null> {
  try {
    const res = await fetch(url, {
      method: 'PUT',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    const json: ApiResponse<T> = await res.json()
    if (json.success) return json.data
    return null
  } catch {
    return null
  }
}

/**
 * DELETE request. Returns `data` on success, `null` on failure.
 */
export async function apiDelete<T = void>(url: string): Promise<T | null> {
  try {
    const res = await fetch(url, {
      method: 'DELETE',
      headers: authHeaders(),
    })
    if (res.status === 204) return null as T
    const json: ApiResponse<T> = await res.json()
    if (json.success) return json.data
    return null
  } catch {
    return null
  }
}

/**
 * Raw fetch with auth headers for non-standard API patterns (SSE, binary, etc.)
 */
export async function authFetch(url: string, init?: RequestInit): Promise<Response> {
  return fetch(url, {
    ...init,
    headers: { ...authHeaders(), ...init?.headers },
  })
}
