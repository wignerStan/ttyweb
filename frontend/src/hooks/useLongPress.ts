import { useCallback, useEffect, useRef } from 'react'

interface LongPressOptions {
  duration?: number
  moveTolerance?: number
}

interface LongPressHandle {
  cancel: () => void
}

export function useLongPress(
  element: HTMLElement | null,
  callback: () => void,
  options: LongPressOptions = {},
): LongPressHandle {
  const { duration = 500, moveTolerance = 10 } = options
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startPosRef = useRef<{ x: number; y: number } | null>(null)
  const callbackRef = useRef(callback)
  callbackRef.current = callback

  const cancel = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    startPosRef.current = null
  }, [])

  useEffect(() => {
    const el = element
    if (!el) return

    const handleTouchStart = (e: TouchEvent) => {
      const touch = e.touches[0]
      if (!touch) return

      startPosRef.current = { x: touch.clientX, y: touch.clientY }
      timerRef.current = setTimeout(() => {
        timerRef.current = null
        startPosRef.current = null
        callbackRef.current()
      }, duration)
    }

    const handleTouchMove = (e: TouchEvent) => {
      if (!startPosRef.current) return

      const touch = e.touches[0]
      if (!touch) return

      const dx = touch.clientX - startPosRef.current.x
      const dy = touch.clientY - startPosRef.current.y
      if (Math.sqrt(dx * dx + dy * dy) > moveTolerance) {
        cancel()
      }
    }

    const handleTouchEnd = () => {
      cancel()
    }

    const handleTouchCancel = () => {
      cancel()
    }

    el.addEventListener('touchstart', handleTouchStart, { passive: true })
    el.addEventListener('touchmove', handleTouchMove, { passive: true })
    el.addEventListener('touchend', handleTouchEnd, { passive: true })
    el.addEventListener('touchcancel', handleTouchCancel, { passive: true })

    return () => {
      cancel()
      el.removeEventListener('touchstart', handleTouchStart)
      el.removeEventListener('touchmove', handleTouchMove)
      el.removeEventListener('touchend', handleTouchEnd)
      el.removeEventListener('touchcancel', handleTouchCancel)
    }
  }, [element, duration, moveTolerance, cancel])

  return { cancel }
}
