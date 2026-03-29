import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getKeyboardMetrics, isAndroid, isIOS, isMobile, isStandalonePWA } from './platform'

describe('platform', () => {
  const originalNavigator = { ...navigator }
  const originalWindow = { ...window }

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    // Restore navigator properties
    Object.defineProperty(navigator, 'userAgent', {
      value: originalNavigator.userAgent,
      configurable: true,
    })
    Object.defineProperty(navigator, 'platform', {
      value: originalNavigator.platform,
      configurable: true,
    })
    Object.defineProperty(navigator, 'maxTouchPoints', {
      value: originalNavigator.maxTouchPoints,
      configurable: true,
    })
    // Restore window.matchMedia
    if (originalWindow.matchMedia) {
      window.matchMedia = originalWindow.matchMedia
    }
  })

  describe('isIOS', () => {
    it('should detect iPhone', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)',
        configurable: true,
      })

      expect(isIOS()).toBe(true)
    })

    it('should detect iPad', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X)',
        configurable: true,
      })

      expect(isIOS()).toBe(true)
    })

    it('should detect iPod', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (iPod; CPU iPhone OS 16_0 like Mac OS X)',
        configurable: true,
      })

      expect(isIOS()).toBe(true)
    })

    it('should detect iPadOS 13+ (MacIntel with touch)', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
        configurable: true,
      })
      Object.defineProperty(navigator, 'platform', {
        value: 'MacIntel',
        configurable: true,
      })
      Object.defineProperty(navigator, 'maxTouchPoints', {
        value: 5,
        configurable: true,
      })

      expect(isIOS()).toBe(true)
    })

    it('should not detect desktop Mac as iOS', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
        configurable: true,
      })
      Object.defineProperty(navigator, 'platform', {
        value: 'MacIntel',
        configurable: true,
      })
      Object.defineProperty(navigator, 'maxTouchPoints', {
        value: 0,
        configurable: true,
      })

      expect(isIOS()).toBe(false)
    })

    it('should not detect MacIntel with exactly 1 touch point as iOS', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
        configurable: true,
      })
      Object.defineProperty(navigator, 'platform', {
        value: 'MacIntel',
        configurable: true,
      })
      Object.defineProperty(navigator, 'maxTouchPoints', {
        value: 1,
        configurable: true,
      })

      expect(isIOS()).toBe(false)
    })

    it('should not detect Android as iOS', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36',
        configurable: true,
      })

      expect(isIOS()).toBe(false)
    })

    it('should not detect desktop Chrome as iOS', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value:
          'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        configurable: true,
      })

      expect(isIOS()).toBe(false)
    })
  })

  describe('isAndroid', () => {
    it('should detect Android', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36',
        configurable: true,
      })

      expect(isAndroid()).toBe(true)
    })

    it('should detect Android (case insensitive)', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Linux; android 13) AppleWebKit/537.36',
        configurable: true,
      })

      expect(isAndroid()).toBe(true)
    })

    it('should detect Android with Mobile suffix', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value:
          'Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36',
        configurable: true,
      })

      expect(isAndroid()).toBe(true)
    })

    it('should not detect iPhone as Android', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)',
        configurable: true,
      })

      expect(isAndroid()).toBe(false)
    })

    it('should not detect desktop as Android', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (X11; Linux x86_64) Chrome/120.0.0.0',
        configurable: true,
      })

      expect(isAndroid()).toBe(false)
    })
  })

  describe('isMobile', () => {
    it('should return true for iOS', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)',
        configurable: true,
      })

      expect(isMobile()).toBe(true)
    })

    it('should return true for Android', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36',
        configurable: true,
      })

      expect(isMobile()).toBe(true)
    })

    it('should return false for desktop', () => {
      Object.defineProperty(navigator, 'userAgent', {
        value: 'Mozilla/5.0 (X11; Linux x86_64) Chrome/120.0.0.0',
        configurable: true,
      })

      expect(isMobile()).toBe(false)
    })
  })

  describe('isStandalonePWA', () => {
    it('should return true when navigator.standalone is true (iOS Safari)', () => {
      Object.defineProperty(navigator, 'standalone', {
        value: true,
        configurable: true,
      })
      window.matchMedia = vi.fn().mockReturnValue({ matches: false })

      expect(isStandalonePWA()).toBe(true)
    })

    it('should return false when navigator.standalone is false', () => {
      Object.defineProperty(navigator, 'standalone', {
        value: false,
        configurable: true,
      })
      window.matchMedia = vi.fn().mockReturnValue({ matches: false })

      expect(isStandalonePWA()).toBe(false)
    })

    it('should return true for standalone display-mode match', () => {
      delete (navigator as unknown as Record<string, unknown>).standalone
      window.matchMedia = vi.fn().mockImplementation((query: string) => ({
        matches: query === '(display-mode: standalone)',
      }))

      expect(isStandalonePWA()).toBe(true)
    })

    it('should return true for fullscreen display-mode match', () => {
      delete (navigator as unknown as Record<string, unknown>).standalone
      window.matchMedia = vi.fn().mockImplementation((query: string) => ({
        matches: query === '(display-mode: fullscreen)',
      }))

      expect(isStandalonePWA()).toBe(true)
    })

    it('should return false when neither standalone nor fullscreen', () => {
      delete (navigator as unknown as Record<string, unknown>).standalone
      window.matchMedia = vi.fn().mockReturnValue({ matches: false })

      expect(isStandalonePWA()).toBe(false)
    })

    it('should handle missing matchMedia', () => {
      delete (navigator as unknown as Record<string, unknown>).standalone
      window.matchMedia = undefined as unknown as typeof window.matchMedia

      expect(isStandalonePWA()).toBe(false)
    })
  })

  describe('getKeyboardMetrics', () => {
    it('should return zeroed metrics when visualViewport is unavailable', () => {
      Object.defineProperty(window, 'visualViewport', {
        value: undefined,
        configurable: true,
      })

      expect(getKeyboardMetrics()).toEqual({
        keyboardHeight: 0,
        isKeyboardVisible: false,
      })
    })

    it('should detect visible keyboard from visualViewport', () => {
      Object.defineProperty(window, 'visualViewport', {
        value: {
          height: 400,
          offsetTop: 0,
        },
        configurable: true,
      })
      Object.defineProperty(window, 'innerHeight', {
        value: 800,
        configurable: true,
      })

      const metrics = getKeyboardMetrics()

      expect(metrics.keyboardHeight).toBe(400)
      expect(metrics.isKeyboardVisible).toBe(true)
    })

    it('should return zero when keyboard is not visible', () => {
      Object.defineProperty(window, 'visualViewport', {
        value: {
          height: 800,
          offsetTop: 0,
        },
        configurable: true,
      })
      Object.defineProperty(window, 'innerHeight', {
        value: 800,
        configurable: true,
      })

      const metrics = getKeyboardMetrics()

      expect(metrics.keyboardHeight).toBe(0)
      expect(metrics.isKeyboardVisible).toBe(false)
    })

    it('should account for offsetTop in keyboard height calculation', () => {
      Object.defineProperty(window, 'visualViewport', {
        value: {
          height: 700,
          offsetTop: 50,
        },
        configurable: true,
      })
      Object.defineProperty(window, 'innerHeight', {
        value: 800,
        configurable: true,
      })

      const metrics = getKeyboardMetrics()

      // keyboardHeight = max(0, 800 - 700 - 50) = 50
      expect(metrics.keyboardHeight).toBe(50)
      expect(metrics.isKeyboardVisible).toBe(true)
    })

    it('should never return negative keyboard height', () => {
      Object.defineProperty(window, 'visualViewport', {
        value: {
          height: 900,
          offsetTop: 0,
        },
        configurable: true,
      })
      Object.defineProperty(window, 'innerHeight', {
        value: 800,
        configurable: true,
      })

      const metrics = getKeyboardMetrics()

      expect(metrics.keyboardHeight).toBe(0)
      expect(metrics.isKeyboardVisible).toBe(false)
    })
  })
})
