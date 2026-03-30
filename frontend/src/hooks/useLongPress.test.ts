import { act, renderHook } from '@testing-library/react'
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { useLongPress } from './useLongPress'

function createTouchEvent(type: string, touches: Array<{ clientX: number; clientY: number }>): TouchEvent {
  return new TouchEvent(type, {
    touches: touches.map(
      (t) =>
        ({
          clientX: t.clientX,
          clientY: t.clientY,
          identifier: 0,
          target: document.createElement('div'),
        }) as unknown as Touch,
    ),
    bubbles: true,
    cancelable: true,
  })
}

describe('useLongPress', () => {
  let container: HTMLDivElement
  let addSpy: ReturnType<typeof vi.spyOn>
  let removeSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    vi.useFakeTimers()
    container = document.createElement('div')
    document.body.appendChild(container)

    addSpy = vi.spyOn(container, 'addEventListener')
    removeSpy = vi.spyOn(container, 'removeEventListener')
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.removeChild(container)
    vi.restoreAllMocks()
  })

  it('calls callback after 500ms', () => {
    const callback = vi.fn()
    renderHook(() => useLongPress(container, callback))

    act(() => {
      container.dispatchEvent(createTouchEvent('touchstart', [{ clientX: 0, clientY: 0 }]))
    })

    expect(callback).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(500)
    })

    expect(callback).toHaveBeenCalledTimes(1)
  })

  it('does not call callback if touch ends early', () => {
    const callback = vi.fn()
    renderHook(() => useLongPress(container, callback))

    act(() => {
      container.dispatchEvent(createTouchEvent('touchstart', [{ clientX: 0, clientY: 0 }]))
    })

    act(() => {
      vi.advanceTimersByTime(200)
      container.dispatchEvent(createTouchEvent('touchend', []))
    })

    act(() => {
      vi.advanceTimersByTime(500)
    })

    expect(callback).not.toHaveBeenCalled()
  })

  it('cancels on move beyond tolerance', () => {
    const callback = vi.fn()
    renderHook(() => useLongPress(container, callback, { moveTolerance: 10 }))

    act(() => {
      container.dispatchEvent(createTouchEvent('touchstart', [{ clientX: 0, clientY: 0 }]))
    })

    act(() => {
      vi.advanceTimersByTime(200)
      container.dispatchEvent(createTouchEvent('touchmove', [{ clientX: 20, clientY: 0 }]))
    })

    act(() => {
      vi.advanceTimersByTime(500)
    })

    expect(callback).not.toHaveBeenCalled()
  })

  it('does not cancel on move within tolerance', () => {
    const callback = vi.fn()
    renderHook(() => useLongPress(container, callback, { moveTolerance: 10 }))

    act(() => {
      container.dispatchEvent(createTouchEvent('touchstart', [{ clientX: 0, clientY: 0 }]))
    })

    act(() => {
      vi.advanceTimersByTime(200)
      container.dispatchEvent(createTouchEvent('touchmove', [{ clientX: 5, clientY: 3 }]))
    })

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(callback).toHaveBeenCalledTimes(1)
  })

  it('supports custom duration', () => {
    const callback = vi.fn()
    renderHook(() => useLongPress(container, callback, { duration: 1000 }))

    act(() => {
      container.dispatchEvent(createTouchEvent('touchstart', [{ clientX: 0, clientY: 0 }]))
    })

    act(() => {
      vi.advanceTimersByTime(999)
    })

    expect(callback).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(1)
    })

    expect(callback).toHaveBeenCalledTimes(1)
  })

  it('cancel function prevents callback', () => {
    const callback = vi.fn()
    const { result } = renderHook(() => useLongPress(container, callback))

    act(() => {
      container.dispatchEvent(createTouchEvent('touchstart', [{ clientX: 0, clientY: 0 }]))
    })

    act(() => {
      vi.advanceTimersByTime(200)
      result.current.cancel()
    })

    act(() => {
      vi.advanceTimersByTime(500)
    })

    expect(callback).not.toHaveBeenCalled()
  })

  it('registers touch event listeners', () => {
    renderHook(() => useLongPress(container, vi.fn()))

    expect(addSpy).toHaveBeenCalledWith('touchstart', expect.any(Function), { passive: true })
    expect(addSpy).toHaveBeenCalledWith('touchmove', expect.any(Function), { passive: true })
    expect(addSpy).toHaveBeenCalledWith('touchend', expect.any(Function), { passive: true })
    expect(addSpy).toHaveBeenCalledWith('touchcancel', expect.any(Function), { passive: true })
  })

  it('removes listeners on unmount', () => {
    const { unmount } = renderHook(() => useLongPress(container, vi.fn()))

    unmount()

    expect(removeSpy).toHaveBeenCalledWith('touchstart', expect.any(Function))
    expect(removeSpy).toHaveBeenCalledWith('touchmove', expect.any(Function))
    expect(removeSpy).toHaveBeenCalledWith('touchend', expect.any(Function))
    expect(removeSpy).toHaveBeenCalledWith('touchcancel', expect.any(Function))
  })

  it('handles null element gracefully', () => {
    const callback = vi.fn()
    renderHook(() => useLongPress(null, callback))

    // No errors, no listeners added
    expect(addSpy).not.toHaveBeenCalled()
  })
})
