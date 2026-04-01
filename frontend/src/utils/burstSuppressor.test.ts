import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createBurstDetector } from './burstSuppressor'

describe('createBurstDetector', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns false before the burst threshold is reached', () => {
    const detector = createBurstDetector({ count: 3, windowMs: 500 })

    expect(detector()).toBe(false)
    expect(detector()).toBe(false)
  })

  it('returns true when burst threshold is reached within the window', () => {
    const detector = createBurstDetector({ count: 3, windowMs: 500 })

    expect(detector()).toBe(false)
    expect(detector()).toBe(false)
    expect(detector()).toBe(true)
  })

  it('resets after a burst is detected', () => {
    const detector = createBurstDetector({ count: 3, windowMs: 500 })

    detector()
    detector()
    expect(detector()).toBe(true)
    // After reset, two more calls should not trigger burst
    expect(detector()).toBe(false)
    expect(detector()).toBe(false)
  })

  it('evicts timestamps outside the window', () => {
    const detector = createBurstDetector({ count: 3, windowMs: 500 })

    detector()
    detector()

    vi.advanceTimersByTime(600)

    // Old timestamps are evicted; this is the first in the new window
    expect(detector()).toBe(false)
    expect(detector()).toBe(false)
    expect(detector()).toBe(true)
  })

  it('works with count of 2', () => {
    const detector = createBurstDetector({ count: 2, windowMs: 500 })

    expect(detector()).toBe(false)
    expect(detector()).toBe(true)
  })

  it('handles rapid calls beyond the count', () => {
    const detector = createBurstDetector({ count: 3, windowMs: 500 })

    // 5 calls: first 2 pass, 3rd triggers burst (resets), then 2 more pass
    expect(detector()).toBe(false)
    expect(detector()).toBe(false)
    expect(detector()).toBe(true) // resets
    expect(detector()).toBe(false)
    expect(detector()).toBe(false)
  })
})
