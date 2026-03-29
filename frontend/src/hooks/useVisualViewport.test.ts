import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import useVisualViewport from './useVisualViewport'

describe('useVisualViewport', () => {
  let rootEl: HTMLElement
  let listeners: Record<string, EventListener[]>

  function createMockVisualViewport(height: number) {
    const vvListeners: Record<string, EventListener[]> = { resize: [], scroll: [] }
    return {
      height,
      offsetTop: 0,
      width: 375,
      addEventListener: vi.fn((event: string, fn: EventListener) => {
        vvListeners[event]?.push(fn)
      }),
      removeEventListener: vi.fn((event: string, fn: EventListener) => {
        const idx = vvListeners[event]?.indexOf(fn)
        if (idx !== undefined && idx >= 0) vvListeners[event]?.splice(idx, 1)
      }),
      _listeners: vvListeners,
    }
  }

  beforeEach(() => {
    vi.restoreAllMocks()
    vi.useFakeTimers()
    rootEl = document.documentElement
    rootEl.style.removeProperty('--app-height')
    rootEl.style.removeProperty('--vvh')
    rootEl.style.removeProperty('--vv-offset')
    listeners = { orientationchange: [] }
    vi.spyOn(window, 'addEventListener').mockImplementation(
      (event: string, handler: EventListenerOrEventListenerObject) => {
        if (event === 'orientationchange') {
          listeners[event].push(handler as EventListener)
        }
      },
    )
    vi.spyOn(window, 'removeEventListener').mockImplementation(
      (event: string, handler: EventListenerOrEventListenerObject) => {
        if (event === 'orientationchange') {
          const idx = listeners[event].indexOf(handler as EventListener)
          if (idx >= 0) listeners[event].splice(idx, 1)
        }
      },
    )
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('sets initial --app-height from visualViewport', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    renderHook(() => useVisualViewport())

    expect(rootEl.style.getPropertyValue('--app-height')).toBe('800px')
  })

  it('does nothing when visualViewport is not available', () => {
    vi.stubGlobal('visualViewport', undefined)

    renderHook(() => useVisualViewport())

    expect(rootEl.style.getPropertyValue('--app-height')).toBe('')
  })

  it('updates --app-height on resize when change exceeds jitter threshold', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    renderHook(() => useVisualViewport())

    // Simulate keyboard closing: viewport grows from 500 to 790
    vv.height = 790
    const resizeEvent = new Event('resize')
    vv._listeners.resize.forEach((fn) => fn(resizeEvent))

    expect(rootEl.style.getPropertyValue('--app-height')).toBe('790px')
    expect(rootEl.style.getPropertyValue('--vvh')).toBe('790px')
    expect(rootEl.style.getPropertyValue('--vv-offset')).toBe('0px')
  })

  it('freezes layout height when keyboard opens (large drop)', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    renderHook(() => useVisualViewport())

    expect(rootEl.style.getPropertyValue('--app-height')).toBe('800px')

    // Keyboard opens: viewport drops from 800 to 400 (diff = 400 > 150 threshold)
    vv.height = 400
    vv._listeners.resize.forEach((fn) => fn(new Event('resize')))

    // --app-height should remain at 800 (frozen), but --vvh and --vv-offset should update
    expect(rootEl.style.getPropertyValue('--app-height')).toBe('800px')
    expect(rootEl.style.getPropertyValue('--vvh')).toBe('400px')
    expect(rootEl.style.getPropertyValue('--vv-offset')).toBe('400px')
  })

  it('ignores sub-pixel jitter', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    renderHook(() => useVisualViewport())

    // Small change: 800 -> 799 (diff = 1 < 3 jitter threshold)
    vv.height = 799
    vv._listeners.resize.forEach((fn) => fn(new Event('resize')))

    expect(rootEl.style.getPropertyValue('--app-height')).toBe('800px')
  })

  it('handles scroll events', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    renderHook(() => useVisualViewport())

    // Large change via scroll
    vv.height = 600
    vv._listeners.scroll.forEach((fn) => fn(new Event('scroll')))

    // diff = 200 > 150 threshold -> freeze
    expect(rootEl.style.getPropertyValue('--app-height')).toBe('800px')
    expect(rootEl.style.getPropertyValue('--vvh')).toBe('600px')
    expect(rootEl.style.getPropertyValue('--vv-offset')).toBe('200px')
  })

  it('handles orientation change with delayed update', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    renderHook(() => useVisualViewport())

    // Simulate orientation change
    vv.height = 780
    listeners.orientationchange.forEach((fn) => fn(new Event('orientationchange')))

    // Before timeout -- height should still be old value
    expect(rootEl.style.getPropertyValue('--app-height')).toBe('800px')

    // After 300ms timeout: setAppHeight(780) is called
    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(rootEl.style.getPropertyValue('--app-height')).toBe('780px')
  })

  it('removes all event listeners on unmount', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)

    const { unmount } = renderHook(() => useVisualViewport())

    unmount()

    expect(vv.removeEventListener).toHaveBeenCalledWith('resize', expect.any(Function))
    expect(vv.removeEventListener).toHaveBeenCalledWith('scroll', expect.any(Function))
    expect(window.removeEventListener).toHaveBeenCalledWith(
      'orientationchange',
      expect.any(Function),
    )
  })
})
