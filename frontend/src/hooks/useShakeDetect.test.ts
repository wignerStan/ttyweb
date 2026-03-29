import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import useShakeDetect from './useShakeDetect'

// Helper to create a DeviceMotionEvent-like object
function createMotionEvent(acceleration: {
  x: number | null
  y: number | null
  z: number | null
}): DeviceMotionEvent {
  return {
    accelerationIncludingGravity: acceleration,
  } as unknown as DeviceMotionEvent
}

describe('useShakeDetect', () => {
  let motionListeners: Array<(e: DeviceMotionEvent) => void>

  beforeEach(() => {
    vi.restoreAllMocks()
    motionListeners = []
    vi.useFakeTimers()

    // Mock window.addEventListener to capture devicemotion listeners
    vi.spyOn(window, 'addEventListener').mockImplementation(
      (event: string, handler: EventListenerOrEventListenerObject) => {
        if (event === 'devicemotion') {
          motionListeners.push(handler as (e: DeviceMotionEvent) => void)
        }
      },
    )
    vi.spyOn(window, 'removeEventListener').mockImplementation(
      (event: string, handler: EventListenerOrEventListenerObject) => {
        if (event === 'devicemotion') {
          const idx = motionListeners.indexOf(handler as (e: DeviceMotionEvent) => void)
          if (idx >= 0) motionListeners.splice(idx, 1)
        }
      },
    )
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('registers a devicemotion listener when enabled', () => {
    renderHook(() => useShakeDetect(vi.fn()))

    expect(window.addEventListener).toHaveBeenCalledWith('devicemotion', expect.any(Function))
  })

  it('does not register a devicemotion listener when disabled', () => {
    renderHook(() => useShakeDetect(vi.fn(), { enabled: false }))

    expect(window.addEventListener).not.toHaveBeenCalledWith('devicemotion', expect.any(Function))
  })

  it('detects shake above threshold', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake, { threshold: 25, shakeCount: 2 }))

    // First shake event (magnitude ~50 > 25)
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).not.toHaveBeenCalled()

    // Second shake event within window
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).toHaveBeenCalledTimes(1)
  })

  it('ignores acceleration below threshold', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake, { threshold: 25 }))

    // Magnitude ~5 < 25
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 3, y: 2, z: 3 })))
    })

    expect(onShake).not.toHaveBeenCalled()
  })

  it('respects shakeCount option', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake, { threshold: 25, shakeCount: 3 }))

    // Two shakes should not trigger (need 3)
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).not.toHaveBeenCalled()

    // Third shake triggers
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).toHaveBeenCalledTimes(1)
  })

  it('respects cooldown period', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake, { threshold: 25, shakeCount: 2, cooldown: 2000 }))

    // Trigger shake
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).toHaveBeenCalledTimes(1)

    // Immediately try again (within cooldown)
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).toHaveBeenCalledTimes(1)

    // Wait past cooldown
    act(() => {
      vi.advanceTimersByTime(2001)
    })

    // Now it should trigger again
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).toHaveBeenCalledTimes(2)
  })

  it('ignores shakes outside shakeWindow', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake, { threshold: 25, shakeCount: 2, shakeWindow: 500 }))

    // First shake
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    // Advance past shake window
    act(() => {
      vi.advanceTimersByTime(501)
    })

    // Second shake is now outside window, so this is only count=1
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake).not.toHaveBeenCalled()
  })

  it('ignores events with null acceleration values', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake))

    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: null, y: null, z: null })))
    })

    expect(onShake).not.toHaveBeenCalled()
  })

  it('ignores events with no acceleration data', () => {
    const onShake = vi.fn()
    renderHook(() => useShakeDetect(onShake))

    act(() => {
      motionListeners.forEach((fn) =>
        fn({ accelerationIncludingGravity: null } as unknown as DeviceMotionEvent),
      )
    })

    expect(onShake).not.toHaveBeenCalled()
  })

  it('removes devicemotion listener on unmount', () => {
    const { unmount } = renderHook(() => useShakeDetect(vi.fn()))

    unmount()

    expect(window.removeEventListener).toHaveBeenCalledWith('devicemotion', expect.any(Function))
    expect(motionListeners).toHaveLength(0)
  })

  it('updates callback when onShake changes', () => {
    const onShake1 = vi.fn()
    const onShake2 = vi.fn()
    const { rerender } = renderHook(
      ({ cb }) => useShakeDetect(cb, { threshold: 25, shakeCount: 2 }),
      { initialProps: { cb: onShake1 } },
    )

    // Trigger shake with first callback
    act(() => {
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake1).toHaveBeenCalledTimes(1)

    // Rerender with new callback
    rerender({ cb: onShake2 })

    // Advance past cooldown then trigger again
    act(() => {
      vi.advanceTimersByTime(2001)
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
      motionListeners.forEach((fn) => fn(createMotionEvent({ x: 30, y: 20, z: 30 })))
    })

    expect(onShake2).toHaveBeenCalledTimes(1)
    expect(onShake1).toHaveBeenCalledTimes(1) // Not called again
  })

  it('returns requestPermission function', () => {
    const { result } = renderHook(() => useShakeDetect(vi.fn()))

    expect(typeof result.current.requestPermission).toBe('function')
  })
})
