import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// Mock platform utilities at module level so vi.mock hoisting works
const mockIsMobile = vi.fn()
const mockGetKeyboardMetrics = vi.fn()
const mockIsDebugEnabled = vi.fn()

vi.mock('../utils/platform', () => ({
  isMobile: (...args: unknown[]) => mockIsMobile(...args),
  getKeyboardMetrics: (...args: unknown[]) => mockGetKeyboardMetrics(...args),
}))

vi.mock('../utils/telemetry', () => ({
  isDebugEnabled: (...args: unknown[]) => mockIsDebugEnabled(...args),
}))

import { useKeyboardAvoider } from './useKeyboardAvoider'

// Helper to create a mock VisualViewport
function createMockVisualViewport(height: number, offsetTop = 0) {
  const listeners: Record<string, EventListener[]> = { resize: [], scroll: [] }
  return {
    height,
    offsetTop,
    width: 375,
    addEventListener: vi.fn((event: string, fn: EventListener) => {
      listeners[event]?.push(fn)
    }),
    removeEventListener: vi.fn((event: string, fn: EventListener) => {
      const idx = listeners[event]?.indexOf(fn)
      if (idx !== undefined && idx >= 0) listeners[event]?.splice(idx, 1)
    }),
    _listeners: listeners,
  }
}

describe('useKeyboardAvoider', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.useFakeTimers()
    mockIsDebugEnabled.mockReturnValue(false)
    mockIsMobile.mockReturnValue(true)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 0, isKeyboardVisible: false })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns default state when keyboard is not visible', () => {
    const { result } = renderHook(() => useKeyboardAvoider())

    expect(result.current.keyboardHeight).toBe(0)
    expect(result.current.keyboardHeightPx).toBe(0)
    expect(result.current.isKeyboardVisible).toBe(false)
    expect(result.current.keyboardVisible).toBe(false)
    expect(result.current.keyboardSpacerHeightPx).toBe(0)
  })

  it('returns empty containerStyle on desktop (non-mobile)', () => {
    mockIsMobile.mockReturnValue(false)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 300, isKeyboardVisible: true })

    const { result } = renderHook(() => useKeyboardAvoider())

    expect(result.current.containerStyle).toEqual({})
    expect(result.current.keyboardSpacerHeightPx).toBe(0)
  })

  it('returns containerStyle with paddingBottom when keyboard is visible on mobile', () => {
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 300, isKeyboardVisible: true })

    const { result } = renderHook(() => useKeyboardAvoider())

    expect(result.current.keyboardHeight).toBe(300)
    expect(result.current.isKeyboardVisible).toBe(true)
    expect(result.current.keyboardSpacerHeightPx).toBe(300)
    expect(result.current.containerStyle.paddingBottom).toBe('300px')
    expect(result.current.containerStyle.height).toBe('100dvh')
  })

  it('includes accessoryHeight in paddingBottom', () => {
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 250, isKeyboardVisible: true })

    const { result } = renderHook(() => useKeyboardAvoider(true, 50))

    expect(result.current.containerStyle.paddingBottom).toBe('300px')
  })

  it('uses accessoryHeight as paddingBottom when keyboard is hidden', () => {
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 0, isKeyboardVisible: false })

    const { result } = renderHook(() => useKeyboardAvoider(true, 50))

    expect(result.current.containerStyle.paddingBottom).toBe('50px')
  })

  it('debounces metric updates via visualViewport resize', () => {
    const vv = createMockVisualViewport(600)
    vi.stubGlobal('visualViewport', vv)

    let callCount = 0
    mockGetKeyboardMetrics.mockImplementation(() => {
      callCount++
      return { keyboardHeight: 0, isKeyboardVisible: false }
    })

    renderHook(() => useKeyboardAvoider())

    // Initial metrics call
    const countAfterMount = callCount

    // Trigger a resize event
    act(() => {
      vv._listeners.resize!.forEach((fn) => fn(new Event('resize')))
    })

    // Before debounce fires, should not have called again
    expect(callCount).toBe(countAfterMount)

    // After debounce (100ms)
    act(() => {
      vi.advanceTimersByTime(100)
    })

    // Should have called again after debounce
    expect(callCount).toBeGreaterThan(countAfterMount)
  })

  it('does not set state when metrics are unchanged', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 0, isKeyboardVisible: false })

    const { result } = renderHook(() => useKeyboardAvoider())

    const initialValue = result.current.keyboardHeight

    // Trigger resize, then debounce
    act(() => {
      vv._listeners.resize!.forEach((fn) => fn(new Event('resize')))
      vi.advanceTimersByTime(100)
    })

    // Value should remain the same since metrics didn't change
    expect(result.current.keyboardHeight).toBe(initialValue)
  })

  it('removes event listeners on unmount', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 0, isKeyboardVisible: false })

    const { unmount } = renderHook(() => useKeyboardAvoider())

    unmount()

    expect(vv.removeEventListener).toHaveBeenCalledWith('resize', expect.any(Function))
    expect(vv.removeEventListener).toHaveBeenCalledWith('scroll', expect.any(Function))
  })

  it('does not attach listeners when enabled is false', () => {
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 0, isKeyboardVisible: false })

    const { unmount } = renderHook(() => useKeyboardAvoider(false))

    expect(vv.addEventListener).not.toHaveBeenCalled()

    unmount()
  })

  it('exposes window.__keyboardMetrics when debug is enabled and mobile', () => {
    mockIsDebugEnabled.mockReturnValue(true)
    mockIsMobile.mockReturnValue(true)
    const vv = createMockVisualViewport(500)
    vi.stubGlobal('visualViewport', vv)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 300, isKeyboardVisible: true })

    renderHook(() => useKeyboardAvoider())

    expect(window.__keyboardMetrics).toBeDefined()
    expect(window.__keyboardMetrics!()).toEqual({
      keyboardVisible: true,
      keyboardHeightPx: 300,
      keyboardSpacerHeightPx: 300,
      visualViewportHeight: 500,
      visualViewportWidth: 375,
      layoutHeight: window.innerHeight,
      layoutWidth: window.innerWidth,
    })
  })

  it('cleans up window.__keyboardMetrics on unmount', () => {
    mockIsDebugEnabled.mockReturnValue(true)
    mockIsMobile.mockReturnValue(true)
    const vv = createMockVisualViewport(800)
    vi.stubGlobal('visualViewport', vv)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 0, isKeyboardVisible: false })

    const { unmount } = renderHook(() => useKeyboardAvoider())

    expect(window.__keyboardMetrics).toBeDefined()

    unmount()

    expect(window.__keyboardMetrics).toBeUndefined()
  })

  it('does not update metrics when disabled on mobile', () => {
    mockIsMobile.mockReturnValue(true)
    mockGetKeyboardMetrics.mockReturnValue({ keyboardHeight: 300, isKeyboardVisible: true })

    const { result } = renderHook(() => useKeyboardAvoider(false))

    // When explicitly disabled, even on mobile, should return desktop-like state
    expect(result.current.containerStyle).toEqual({})
    expect(result.current.keyboardSpacerHeightPx).toBe(0)
  })
})
