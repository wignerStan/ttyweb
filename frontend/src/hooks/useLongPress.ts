import { useCallback, useRef } from 'react'

interface UseLongPressOptions {
  threshold?: number
  onLongPress: (e: React.TouchEvent | React.MouseEvent) => void
}

export function useLongPress({ threshold = 500, onLongPress }: UseLongPressOptions) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isLongPressRef = useRef(false)
  const movedRef = useRef(false)

  const clear = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
    movedRef.current = false
  }, [])

  const start = useCallback(
    (e: React.TouchEvent | React.MouseEvent) => {
      // Ignore multi-touch events — only process single-finger gestures
      if ('touches' in e && e.touches.length !== 1) {
        return
      }

      isLongPressRef.current = false
      movedRef.current = false
      timerRef.current = setTimeout(() => {
        if (movedRef.current) return
        isLongPressRef.current = true
        onLongPress(e)
      }, threshold)
    },
    [threshold, onLongPress],
  )

  const move = useCallback(() => {
    movedRef.current = true
    clear()
  }, [clear])

  const cancel = useCallback(
    (e: React.TouchEvent | React.MouseEvent) => {
      // After a long press, prevent the synthetic click that mobile browsers fire
      if (isLongPressRef.current && 'preventDefault' in e) {
        e.preventDefault()
      }
      clear()
    },
    [clear],
  )

  return {
    onTouchStart: start,
    onTouchEnd: cancel,
    onTouchMove: move,
    onMouseDown: start,
    onMouseUp: cancel,
    onMouseLeave: cancel,
    isLongPressRef,
  }
}
