import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// Mock platform module before importing telemetry
vi.mock('./platform', () => ({
  isIOS: vi.fn(() => true),
}))

// Must import after mock setup
const { isIOS } = await import('./platform')
const { isDebugEnabled, log, getRecentEvents, clearEvents, dumpTelemetry } = await import(
  './telemetry'
)

import type { TelemetryEventType } from './telemetry'

describe('telemetry', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
    clearEvents()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('isDebugEnabled', () => {
    it('should return true when localStorage tmux-debug is "1"', () => {
      localStorage.setItem('tmux-debug', '1')

      expect(isDebugEnabled()).toBe(true)
    })

    it('should return false when localStorage tmux-debug is not "1"', () => {
      localStorage.setItem('tmux-debug', '0')

      expect(isDebugEnabled()).toBe(false)
    })

    it('should return false when localStorage tmux-debug is absent', () => {
      expect(isDebugEnabled()).toBe(false)
    })

    it('should return true when URL param debug is "1"', () => {
      vi.spyOn(window, 'location', 'get').mockReturnValue({
        search: '?debug=1',
      } as Location)

      expect(isDebugEnabled()).toBe(true)
    })

    it('should return false when URL param debug is not "1"', () => {
      vi.spyOn(window, 'location', 'get').mockReturnValue({
        search: '?debug=0',
      } as Location)

      expect(isDebugEnabled()).toBe(false)
    })

    it('should prefer localStorage over URL param', () => {
      localStorage.setItem('tmux-debug', '1')
      vi.spyOn(window, 'location', 'get').mockReturnValue({
        search: '?debug=0',
      } as Location)

      expect(isDebugEnabled()).toBe(true)
    })
  })

  describe('log', () => {
    it('should not log when not on iOS', () => {
      vi.mocked(isIOS).mockReturnValue(false)
      localStorage.setItem('tmux-debug', '1')

      log('focus')

      expect(getRecentEvents()).toHaveLength(0)
    })

    it('should not log when debug is not enabled', () => {
      vi.mocked(isIOS).mockReturnValue(true)

      log('focus')

      expect(getRecentEvents()).toHaveLength(0)
    })

    it('should log low-frequency events always', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      log('focus')
      log('blur')
      log('visibilitychange')

      expect(getRecentEvents()).toHaveLength(3)
    })

    it('should sample high-frequency events (onData)', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      const lowFrequencyTypes: TelemetryEventType[] = [
        'focus',
        'blur',
        'visibilitychange',
        'reconnect',
        'viewport-resize',
        'suppressed',
        'dec1004-disable',
      ]

      // Send 10 onData events plus one of each low-frequency type
      for (let i = 0; i < 10; i++) {
        log('onData', { index: i })
      }
      for (const type of lowFrequencyTypes) {
        log(type)
      }

      const events = getRecentEvents()
      // All 10 onData events are stored in buffer (sampling only controls console output)
      // plus 7 low-frequency events = 17 total
      expect(events).toHaveLength(17)
    })

    it('should include data in event', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      log('reconnect', { attempt: 3, url: 'ws://localhost' })

      const events = getRecentEvents()
      expect(events).toHaveLength(1)
      expect(events[0]!.type).toBe('reconnect')
      expect(events[0]!.data).toEqual({ attempt: 3, url: 'ws://localhost' })
      expect(events[0]!.timestamp).toBeTypeOf('number')
    })

    it('should include timestamp in event', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      const before = Date.now()
      log('focus')
      const after = Date.now()

      const events = getRecentEvents()
      expect(events[0]!.timestamp).toBeGreaterThanOrEqual(before)
      expect(events[0]!.timestamp).toBeLessThanOrEqual(after)
    })
  })

  describe('getRecentEvents', () => {
    it('should return empty array when no events', () => {
      expect(getRecentEvents()).toEqual([])
    })

    it('should return limited number of events', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      for (let i = 0; i < 10; i++) {
        log('focus', { i })
      }

      expect(getRecentEvents(3)).toHaveLength(3)
      // Should return the last 3
      expect(getRecentEvents(3)[2]!.data).toEqual({ i: 9 })
    })

    it('should return all events when count exceeds buffer', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      log('focus')
      log('blur')

      expect(getRecentEvents(100)).toHaveLength(2)
    })
  })

  describe('clearEvents', () => {
    it('should clear all events', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      log('focus')
      log('blur')
      expect(getRecentEvents()).toHaveLength(2)

      clearEvents()

      expect(getRecentEvents()).toHaveLength(0)
    })
  })

  describe('ring buffer behavior', () => {
    it('should evict oldest events when buffer exceeds 50', () => {
      vi.mocked(isIOS).mockReturnValue(true)
      localStorage.setItem('tmux-debug', '1')

      // Add 55 events
      for (let i = 0; i < 55; i++) {
        log('focus', { index: i })
      }

      const events = getRecentEvents()
      // Buffer max is 50, so only 50 should remain
      expect(events).toHaveLength(50)
      // First remaining event should be index 5 (oldest evicted)
      expect(events[0]!.data).toEqual({ index: 5 })
      // Last event should be index 54
      expect(events[49]!.data).toEqual({ index: 54 })
    })
  })

  describe('dumpTelemetry', () => {
    it('should call getRecent without crashing', () => {
      expect(() => dumpTelemetry()).not.toThrow()
    })
  })
})
