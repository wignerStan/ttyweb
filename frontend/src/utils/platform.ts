/**
 * Platform detection utilities.
 * Pure functions with no side effects for iOS/Android/PWA detection.
 */

/** Detect iOS devices including iPadOS (iPadOS 13+ reports as "MacIntel" but has touch). */
export function isIOS(): boolean {
  if (typeof navigator === 'undefined') return false

  const ua = navigator.userAgent

  if (/iPhone|iPad|iPod/.test(ua)) {
    return true
  }

  // iPadOS 13+: reports as Mac but has touch
  if (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1) {
    return true
  }

  return false
}

/** Detect Android devices. */
export function isAndroid(): boolean {
  if (typeof navigator === 'undefined') return false
  return /Android/i.test(navigator.userAgent)
}

/** Detect if running as installed PWA (standalone mode). */
export function isStandalonePWA(): boolean {
  if (typeof window === 'undefined') return false

  if ('standalone' in navigator && (navigator as { standalone?: boolean }).standalone) {
    return true
  }

  if (window.matchMedia?.('(display-mode: standalone)').matches) {
    return true
  }

  if (window.matchMedia?.('(display-mode: fullscreen)').matches) {
    return true
  }

  return false
}

/** Check if device is mobile (iOS or Android). */
export function isMobile(): boolean {
  return isIOS() || isAndroid()
}

export interface KeyboardMetrics {
  keyboardHeight: number
  isKeyboardVisible: boolean
}

export function getKeyboardMetrics(): KeyboardMetrics {
  const vv = window.visualViewport
  if (!vv) return { keyboardHeight: 0, isKeyboardVisible: false }
  const keyboardHeight = Math.max(0, window.innerHeight - vv.height - vv.offsetTop)
  const isKeyboardVisible = keyboardHeight > 0
  return { keyboardHeight, isKeyboardVisible }
}
