import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createTelemetryEmitter } from './telemetryEmitter'

describe('telemetryEmitter', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal('localStorage', { getItem: () => null })
    Object.defineProperty(document, 'hidden', { value: false, writable: true, configurable: true })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  describe('debug disabled', () => {
    it('emit is a no-op', () => {
      const emitter = createTelemetryEmitter('p1')
      expect(() => emitter.emit('mobile-onData')).not.toThrow()
    })

    it('flush is a no-op', () => {
      const emitter = createTelemetryEmitter('p1')
      expect(() => emitter.flush()).not.toThrow()
    })

    it('destroy is a no-op', () => {
      const emitter = createTelemetryEmitter('p1')
      expect(() => emitter.destroy()).not.toThrow()
    })
  })

  describe('debug enabled', () => {
    beforeEach(() => {
      vi.stubGlobal('localStorage', {
        getItem: (key: string) => (key === 'tmux-debug' ? '1' : null),
      })
    })

    it('emit batches events', () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', fetchMock)

      const emitter = createTelemetryEmitter('p1')
      emitter.emit('mobile-onData')
      emitter.emit('touch-start')
      emitter.flush()
      expect(fetchMock).toHaveBeenCalledTimes(1)

      const callArgs = fetchMock.mock.calls[0]
      expect(callArgs).toBeDefined()
      const [, options] = callArgs!
      const body = JSON.parse(options.body)
      expect(body.events).toHaveLength(2)
      expect(body.events[0].event).toBe('mobile-onData')
      expect(body.events[1].event).toBe('touch-start')

      emitter.destroy()
    })

    it('flush sends batched events via POST', async () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', fetchMock)

      const emitter = createTelemetryEmitter('p2')
      emitter.emit('mobile-onData', { data: 'test' })
      emitter.flush()

      expect(fetchMock).toHaveBeenCalledTimes(1)
      const callArgs = fetchMock.mock.calls[0]
      expect(callArgs).toBeDefined()
      const [, options] = callArgs!
      expect(options.method).toBe('POST')
      const body = JSON.parse(options.body)
      expect(body.events).toHaveLength(1)
      expect(body.events[0].event).toBe('mobile-onData')
      expect(body.events[0].paneId).toBe('p2')

      emitter.destroy()
    })

    it('auto-flushes when batch reaches threshold', () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', fetchMock)

      const emitter = createTelemetryEmitter('p3')
      for (let i = 0; i < 50; i++) {
        emitter.emit('mobile-onData')
      }
      // Batch size threshold is 50, should auto-flush at exactly 50
      expect(fetchMock).toHaveBeenCalledTimes(1)

      emitter.emit('mobile-onData')
      // Only 1 more event, not enough to trigger another flush
      expect(fetchMock).toHaveBeenCalledTimes(1)

      emitter.destroy()
    })

    it('flush on interval timer', () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', fetchMock)

      const emitter = createTelemetryEmitter('p4')
      emitter.emit('mobile-onData')

      vi.advanceTimersByTime(1000)
      expect(fetchMock).toHaveBeenCalledTimes(1)

      emitter.destroy()
    })

    it('destroy cleans up timers and event listeners', () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', fetchMock)
      const removeSpy = vi.spyOn(document, 'removeEventListener')
      const windowSpy = vi.spyOn(window, 'removeEventListener')

      const emitter = createTelemetryEmitter('p5')
      emitter.emit('mobile-onData')
      emitter.destroy()

      expect(removeSpy).toHaveBeenCalledWith('visibilitychange', expect.any(Function))
      expect(windowSpy).toHaveBeenCalledWith('beforeunload', expect.any(Function))

      // After destroy, further flushes should not fetch
      fetchMock.mockClear()
      emitter.emit('mobile-onData')
      vi.advanceTimersByTime(2000)
      expect(fetchMock).not.toHaveBeenCalled()
    })

    it('truncates long string data', async () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      vi.stubGlobal('fetch', fetchMock)

      const emitter = createTelemetryEmitter('p6')
      const longData = 'x'.repeat(200)
      emitter.emit('mobile-onData', { data: longData })
      emitter.flush()

      const callArgs = fetchMock.mock.calls[0]
      expect(callArgs).toBeDefined()
      const [, options] = callArgs!
      const body = JSON.parse(options.body)
      const sentData = body.events[0]!.data as string
      expect(sentData.length).toBeLessThan(200)
      expect(sentData).toContain('[truncated]')

      emitter.destroy()
    })

    it('uses sendBeacon on visibility change to hidden', () => {
      const beaconMock = vi.fn().mockReturnValue(true)
      vi.stubGlobal('navigator', { ...navigator, sendBeacon: beaconMock })

      const emitter = createTelemetryEmitter('p7')
      emitter.emit('mobile-onData')

      Object.defineProperty(document, 'hidden', { value: true, writable: true, configurable: true })
      document.dispatchEvent(new Event('visibilitychange'))

      expect(beaconMock).toHaveBeenCalledTimes(1)

      emitter.destroy()
    })
  })
})
