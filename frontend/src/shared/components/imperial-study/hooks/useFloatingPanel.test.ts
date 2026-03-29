import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useFloatingPanel } from './useFloatingPanel'

beforeEach(() => {
  localStorage.clear()
  Object.defineProperty(window, 'innerWidth', { value: 1280, writable: true })
  Object.defineProperty(window, 'innerHeight', { value: 720, writable: true })
})

afterEach(() => {
  vi.restoreAllMocks()
  Object.defineProperty(window, 'innerWidth', { value: 1024, writable: true })
  Object.defineProperty(window, 'innerHeight', { value: 768, writable: true })
})

describe('useFloatingPanel', () => {
  it('returns default state values', () => {
    const { result } = renderHook(() => useFloatingPanel())
    expect(result.current.collapsed).toBe(false)
    expect(result.current.opacity).toBe(0.15)
    expect(result.current.size).toEqual({ width: 860, height: 280 })
    // Auto position: x = max(16, (1280 - 860) / 2) = 210, y = 720 - 280 - 16 = 424
    expect(result.current.position).toEqual({ x: 210, y: 424 })
  })

  it('toggleCollapse toggles collapsed state', () => {
    const { result } = renderHook(() => useFloatingPanel())
    expect(result.current.collapsed).toBe(false)
    act(() => result.current.toggleCollapse())
    expect(result.current.collapsed).toBe(true)
    act(() => result.current.toggleCollapse())
    expect(result.current.collapsed).toBe(false)
  })

  it('setCollapsed and setOpacity update state', () => {
    const { result } = renderHook(() => useFloatingPanel())
    act(() => result.current.setCollapsed(true))
    expect(result.current.collapsed).toBe(true)
    act(() => result.current.setOpacity(0.5))
    expect(result.current.opacity).toBe(0.5)
  })

  it('resolves auto position (-1, -1) to bottom-center', () => {
    const { result } = renderHook(() => useFloatingPanel())
    const pos = result.current.position
    expect(pos.x).toBe(210)
    expect(pos.y).toBe(424)
  })

  it('respects explicit position', () => {
    const { result } = renderHook(() => useFloatingPanel({ defaultPosition: { x: 100, y: 50 } }))
    expect(result.current.position).toEqual({ x: 100, y: 50 })
  })

  it('persists state to localStorage', () => {
    const { result } = renderHook(() => useFloatingPanel({ storageKey: 'test-panel' }))
    act(() => result.current.setCollapsed(true))
    act(() => result.current.setOpacity(0.8))
    expect(localStorage.getItem('test-panel-collapsed')).toBe('true')
    expect(localStorage.getItem('test-panel-opacity')).toBe('0.8')
  })

  it('loads state from localStorage', () => {
    localStorage.setItem('test-load-collapsed', 'true')
    localStorage.setItem('test-load-opacity', '0.6')
    const { result } = renderHook(() => useFloatingPanel({ storageKey: 'test-load' }))
    expect(result.current.collapsed).toBe(true)
    expect(result.current.opacity).toBe(0.6)
  })

  it('onDragStart updates position on mouse events', () => {
    const addListenerSpy = vi.spyOn(document, 'addEventListener')
    const { result } = renderHook(() => useFloatingPanel())

    const startEvent = {
      preventDefault: vi.fn(),
      clientX: 200,
      clientY: 400,
    } as unknown as React.MouseEvent
    act(() => result.current.onDragStart(startEvent))

    const moveCall = addListenerSpy.mock.calls.find((c) => c[0] === 'mousemove')
    expect(moveCall).toBeDefined()

    const moveHandler = moveCall![1] as (e: MouseEvent) => void
    act(() => {
      moveHandler({ clientX: 300, clientY: 500 } as unknown as MouseEvent)
    })
    // dx=100, dy=100; startX was 210 (auto), so newX=310
    // dy=100, clamp: min(720-40, 424+100) = min(680, 524) = 524
    expect(result.current.position.x).toBe(310)
    expect(result.current.position.y).toBe(524)
  })

  it('onResizeStart respects min dimensions', () => {
    const addListenerSpy = vi.spyOn(document, 'addEventListener')
    const { result } = renderHook(() => useFloatingPanel({ minWidth: 300, minHeight: 200 }))

    const startEvent = {
      preventDefault: vi.fn(),
      stopPropagation: vi.fn(),
      clientX: 0,
      clientY: 0,
    } as unknown as React.MouseEvent
    act(() => result.current.onResizeStart(startEvent))

    const moveCall = addListenerSpy.mock.calls.find((c) => c[0] === 'mousemove')
    const moveHandler = moveCall![1] as (e: MouseEvent) => void

    // dw = -500, so new width = max(300, 860 - 500) = max(300, 360) = 360
    act(() => {
      moveHandler({ clientX: -500, clientY: -500 } as unknown as MouseEvent)
    })
    expect(result.current.size.width).toBe(360)
    expect(result.current.size.height).toBe(200)
  })
})
