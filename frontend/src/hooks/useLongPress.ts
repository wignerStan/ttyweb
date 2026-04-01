import { useCallback, useRef } from 'react'

interface UseLongPressOptions<T = React.TouchEvent | React.MouseEvent> {
  threshold?: number
  onLongPress: (e: T) => void
  /** Transform the event before passing to onLongPress */
  transform?: (e: React.TouchEvent | React.MouseEvent) => T
}

export function useLongPress<T = React.TouchEvent | React.MouseEvent>({
  threshold = 500,
  onLongPress,
  transform,
}: UseLongPressOptions<T>) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isLongPressRef = useRef(false)
  const firedRef = useRef(false)

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
      firedRef.current = false
      timerRef.current = setTimeout(() => {
        isLongPressRef.current = true
        firedRef.current = true
        onLongPress(transform ? transform(e) : (e as T))
      }, threshold)
    },
    [threshold, onLongPress, transform],
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
    onMouseMove: cancel,
    onMouseLeave: cancel,
    isLongPressRef,
    firedRef,
  }
}
