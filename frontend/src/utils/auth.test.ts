import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { checkAuth, getAuthHeader, getToken, login, logout } from './auth'

const AUTH_KEY = 'ttyweb_auth'

describe('auth', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('login', () => {
    it('should login and store credentials on success', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({}),
      } as Response)

      const result = await login('user', 'pass')

      expect(result).toEqual({ success: true })
      const stored = localStorage.getItem(AUTH_KEY)
      expect(stored).toBeTruthy()
      expect(stored).toBe(btoa('user:pass'))
    })

    it('should return error with server message on 401', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Bad credentials' }),
      } as Response)

      const result = await login('user', 'bad')

      expect(result).toEqual({ success: false, error: 'Bad credentials' })
      expect(localStorage.getItem(AUTH_KEY)).toBeNull()
    })

    it('should return default error message when server provides none', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: false,
        status: 403,
        json: async () => ({ success: false }),
      } as Response)

      const result = await login('user', 'pass')

      expect(result).toEqual({ success: false, error: 'Invalid credentials' })
    })

    it('should return connection failed error on network error', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

      const result = await login('user', 'pass')

      expect(result).toEqual({ success: false, error: 'Connection failed' })
      expect(localStorage.getItem(AUTH_KEY)).toBeNull()
    })

    it('should encode username:password as base64', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({}),
      } as Response)

      await login('alice@example.com', 'p@ss:word')

      const stored = localStorage.getItem(AUTH_KEY)
      expect(stored).toBe(btoa('alice@example.com:p@ss:word'))
    })
  })

  describe('logout', () => {
    it('should remove credentials from localStorage', async () => {
      localStorage.setItem(AUTH_KEY, 'some-creds')

      await logout()

      expect(localStorage.getItem(AUTH_KEY)).toBeNull()
    })

    it('should do nothing if no credentials stored', async () => {
      await logout()

      expect(localStorage.getItem(AUTH_KEY)).toBeNull()
    })
  })

  describe('checkAuth', () => {
    it('should return true when auth is not enabled (auth_enabled: false)', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: { auth_enabled: false } }),
      } as Response)

      const result = await checkAuth()

      expect(result).toBe(true)
    })

    it('should return true when auth is enabled and credentials present', async () => {
      localStorage.setItem(AUTH_KEY, 'valid-creds')
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: { auth_enabled: true } }),
      } as Response)

      const result = await checkAuth()

      expect(result).toBe(true)
    })

    it('should return false when auth is enabled but no credentials', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: { auth_enabled: true } }),
      } as Response)

      const result = await checkAuth()

      expect(result).toBe(false)
    })

    it('should return false when server returns non-ok response', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({ error: 'Unauthorized' }),
      } as Response)

      const result = await checkAuth()

      expect(result).toBe(false)
    })

    it('should return false on network error', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))

      const result = await checkAuth()

      expect(result).toBe(false)
    })

    it('should send Authorization header when credentials exist', async () => {
      localStorage.setItem(AUTH_KEY, 'my-creds')
      const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: { auth_enabled: false } }),
      } as Response)

      await checkAuth()

      expect(fetchSpy).toHaveBeenCalledWith('/api/auth/check', {
        headers: { Authorization: 'Basic my-creds' },
      })
    })

    it('should not send Authorization header when no credentials', async () => {
      const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ success: true, data: { auth_enabled: false } }),
      } as Response)

      await checkAuth()

      expect(fetchSpy).toHaveBeenCalledWith('/api/auth/check', {
        headers: {},
      })
    })
  })

  describe('getAuthHeader', () => {
    it('should return Basic header when credentials exist', () => {
      localStorage.setItem(AUTH_KEY, 'some-creds')

      expect(getAuthHeader()).toBe('Basic some-creds')
    })

    it('should return null when no credentials', () => {
      expect(getAuthHeader()).toBeNull()
    })
  })

  describe('getToken', () => {
    it('should return credentials when present', () => {
      localStorage.setItem(AUTH_KEY, 'my-token')

      expect(getToken()).toBe('my-token')
    })

    it('should return empty string when no credentials', () => {
      expect(getToken()).toBe('')
    })
  })
})
