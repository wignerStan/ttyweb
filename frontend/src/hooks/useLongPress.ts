import { useCallback, useRef } from 'react'

interface UseLongPressOptions {
  threshold?: number
  onLongPress: (e: React.TouchEvent | React.MouseEvent) => void
}

export function useLongPress({ threshold = 500, onLongPress }: UseLongPressOptions) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isLongPressRef = useRef(false)

  const clear = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }, [])

  const start = useCallback(
    (e: React.TouchEvent | React.MouseEvent) => {
      // Ignore multi-touch events — only process single-finger gestures
      if ('touches' in e && e.touches.length !== 1) {
        return
      }

      isLongPressRef.current = false
      timerRef.current = setTimeout(() => {
        isLongPressRef.current = true
        onLongPress(e)
      }, threshold)
    },
    [threshold, onLongPress],
  )

  const cancel = useCallback(() => {
    clear()
  }, [clear])

  return {
    onTouchStart: start,
    onTouchEnd: cancel,
    onTouchMove: cancel,
    onMouseDown: start,
    onMouseUp: cancel,
    onMouseLeave: cancel,
    isLongPressRef,
  }
}
