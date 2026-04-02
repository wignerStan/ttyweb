/**
 * Sliding-window burst detector that suppresses input when a character
 * exceeds a configurable count within a time window.
 *
 * Used to prevent duplicate key events on mobile keyboards (iOS/Android)
 * where the OS may fire rapid repeated keypress events for space, enter, etc.
 */

interface BurstConfig {
  /** Number of occurrences within the window that triggers suppression */
  count: number
  /** Time window in milliseconds */
  windowMs: number
}

export function createBurstDetector(config: BurstConfig) {
  const timestamps: number[] = []

  return function checkBurst(): boolean {
    const now = Date.now()
    timestamps.push(now)

    // Evict timestamps outside the window
    while (timestamps.length > 0 && now - (timestamps[0] ?? 0) > config.windowMs) {
      timestamps.shift()
    }

    if (timestamps.length >= config.count) {
      timestamps.length = 0
      return true
    }

    return false
  }
}
