import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import rlog from './rlog'

describe('rlog', () => {
  let consoleSpy: {
    log: ReturnType<typeof vi.spyOn>
    warn: ReturnType<typeof vi.spyOn>
    error: ReturnType<typeof vi.spyOn>
  }
  let fetchSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    vi.restoreAllMocks()
    consoleSpy = {
      log: vi.spyOn(console, 'log').mockImplementation(() => {}),
      warn: vi.spyOn(console, 'warn').mockImplementation(() => {}),
      error: vi.spyOn(console, 'error').mockImplementation(() => {}),
    }
    fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      status: 200,
    } as Response)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should send info level log', () => {
    rlog.info('test message', { key: 'value' })

    expect(consoleSpy.log).toHaveBeenCalledWith('[Remote] test message', { key: 'value' })
    expect(fetchSpy).toHaveBeenCalledTimes(1)
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/log',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })

  it('should send warn level log', () => {
    rlog.warn('warning message')

    expect(consoleSpy.warn).toHaveBeenCalledWith('[Remote] warning message', '')
    expect(fetchSpy).toHaveBeenCalledTimes(1)
  })

  it('should send error level log', () => {
    rlog.error('error occurred', { code: 500 })

    expect(consoleSpy.error).toHaveBeenCalledWith('[Remote] error occurred', { code: 500 })
    expect(fetchSpy).toHaveBeenCalledTimes(1)
  })

  it('should include correct level in request body', () => {
    rlog.info('info msg')
    rlog.warn('warn msg')
    rlog.error('error msg')

    const calls = fetchSpy.mock.calls
    expect(JSON.parse(calls[0][1].body)).toMatchObject({ level: 'info' })
    expect(JSON.parse(calls[1][1].body)).toMatchObject({ level: 'warn' })
    expect(JSON.parse(calls[2][1].body)).toMatchObject({ level: 'error' })
  })

  it('should include message and data in request body', () => {
    rlog.info('hello world', { foo: 'bar' })

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
    expect(body.message).toBe('hello world')
    expect(body.data).toEqual({ foo: 'bar' })
  })

  it('should truncate long user agent strings to 80 chars with ellipsis', () => {
    const longUA = 'a'.repeat(120)
    Object.defineProperty(navigator, 'userAgent', {
      value: longUA,
      configurable: true,
    })

    rlog.info('test')

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
    expect(body.ua).toBe('a'.repeat(80) + '\u2026')
    expect(body.ua.length).toBe(81)
  })

  it('should not truncate short user agent strings', () => {
    const shortUA = 'Mozilla/5.0 (Linux; Android 13)'
    Object.defineProperty(navigator, 'userAgent', {
      value: shortUA,
      configurable: true,
    })

    rlog.info('test')

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
    expect(body.ua).toBe(shortUA)
  })

  it('should include current URL in request body', () => {
    rlog.info('test')

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
    expect(body.url).toBe(window.location.href)
  })

  it('should send data as undefined when not provided', () => {
    rlog.info('no data')

    const body = JSON.parse(fetchSpy.mock.calls[0][1].body)
    expect(body.data).toBeUndefined()
  })

  it('should handle network error silently (fetch rejects)', () => {
    fetchSpy.mockRejectedValue(new Error('Network failure'))

    expect(() => rlog.error('network fail')).not.toThrow()
  })

  it('should handle fetch throwing synchronously', () => {
    fetchSpy.mockImplementation(() => {
      throw new Error('Sync error')
    })

    expect(() => rlog.error('sync fail')).not.toThrow()
  })
})
