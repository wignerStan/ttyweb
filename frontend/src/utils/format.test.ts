import { describe, expect, it } from 'vitest'
import { formatDuration, formatRelativeTime, formatSize } from './format'

describe('formatSize', () => {
  it('returns "0 B" for zero bytes', () => {
    expect(formatSize(0)).toBe('0 B')
  })

  it('formats bytes without decimal when >= 10', () => {
    expect(formatSize(10)).toBe('10 B')
    expect(formatSize(999)).toBe('999 B')
    expect(formatSize(512)).toBe('512 B')
  })

  it('formats bytes with one decimal when < 10', () => {
    expect(formatSize(1)).toBe('1.0 B')
    expect(formatSize(5)).toBe('5.0 B')
    expect(formatSize(9)).toBe('9.0 B')
  })

  it('formats kilobytes', () => {
    expect(formatSize(1024)).toBe('1.0 KB')
    expect(formatSize(1536)).toBe('1.5 KB')
    expect(formatSize(10240)).toBe('10 KB')
    expect(formatSize(1048576 - 1)).toBe('1024 KB')
  })

  it('formats megabytes', () => {
    expect(formatSize(1024 * 1024)).toBe('1.0 MB')
    expect(formatSize(1024 * 1024 * 5)).toBe('5.0 MB')
    expect(formatSize(1024 * 1024 * 10)).toBe('10 MB')
    expect(formatSize(1024 * 1024 * 100)).toBe('100 MB')
  })

  it('formats gigabytes', () => {
    expect(formatSize(1024 ** 3)).toBe('1.0 GB')
    expect(formatSize(1024 ** 3 * 2.5)).toBe('2.5 GB')
    expect(formatSize(1024 ** 3 * 15)).toBe('15 GB')
  })

  it('formats terabytes', () => {
    expect(formatSize(1024 ** 4)).toBe('1.0 TB')
    expect(formatSize(1024 ** 4 * 3)).toBe('3.0 TB')
    expect(formatSize(1024 ** 4 * 10)).toBe('10 TB')
  })

  it('handles negative input by returning "0 B"', () => {
    expect(formatSize(-1)).toBe('0 B')
    expect(formatSize(-999)).toBe('0 B')
  })

  it('handles petabyte-range values', () => {
    const pb = 1024 ** 5
    expect(formatSize(pb)).toBe('1.0 PB')
    expect(formatSize(pb * 3)).toBe('3.0 PB')
    expect(formatSize(pb * 10)).toBe('10 PB')
  })
})

describe('formatRelativeTime', () => {
  it('returns "just now" for less than 60 seconds ago', () => {
    const now = Date.now()
    expect(formatRelativeTime(now)).toBe('just now')
    expect(formatRelativeTime(now - 30000)).toBe('just now') // 30s ago
    expect(formatRelativeTime(now - 59000)).toBe('just now') // 59s ago
  })

  it('returns minutes ago for less than 60 minutes', () => {
    const now = Date.now()
    expect(formatRelativeTime(now - 60000)).toBe('1m ago') // 1m
    expect(formatRelativeTime(now - 300000)).toBe('5m ago') // 5m
    expect(formatRelativeTime(now - 3540000)).toBe('59m ago') // 59m
  })

  it('returns hours ago for less than 24 hours', () => {
    const now = Date.now()
    expect(formatRelativeTime(now - 3600000)).toBe('1h ago') // 1h
    expect(formatRelativeTime(now - 7200000)).toBe('2h ago') // 2h
    expect(formatRelativeTime(now - 82800000)).toBe('23h ago') // 23h
  })

  it('returns days ago for less than 30 days', () => {
    const now = Date.now()
    expect(formatRelativeTime(now - 86400000)).toBe('1d ago') // 1d
    expect(formatRelativeTime(now - 86400000 * 7)).toBe('7d ago') // 7d
    expect(formatRelativeTime(now - 86400000 * 29)).toBe('29d ago') // 29d
  })

  it('returns locale date string for 30+ days ago', () => {
    const now = Date.now()
    const thirtyDaysAgo = now - 86400000 * 30
    const result = formatRelativeTime(thirtyDaysAgo)
    // Should be a date string from toLocaleDateString(), not "Xd ago"
    expect(result).not.toMatch(/\d+d ago/)
    expect(result).toBe(new Date(thirtyDaysAgo).toLocaleDateString())
  })
})

describe('formatDuration', () => {
  it('formats seconds only (< 60)', () => {
    expect(formatDuration(0)).toBe('0s')
    expect(formatDuration(1)).toBe('1s')
    expect(formatDuration(30)).toBe('30s')
    expect(formatDuration(59)).toBe('59s')
  })

  it('formats minutes and seconds (< 60 min)', () => {
    expect(formatDuration(60)).toBe('1m 0s')
    expect(formatDuration(90)).toBe('1m 30s')
    expect(formatDuration(119)).toBe('1m 59s')
    expect(formatDuration(300)).toBe('5m 0s')
    expect(formatDuration(3599)).toBe('59m 59s')
  })

  it('formats hours and minutes (>= 60 min)', () => {
    expect(formatDuration(3600)).toBe('1h 0m')
    expect(formatDuration(3661)).toBe('1h 1m')
    expect(formatDuration(7384)).toBe('2h 3m') // 2h 3m 4s -> rounds to 2h 3m
    expect(formatDuration(86400)).toBe('24h 0m')
  })

  it('handles zero correctly', () => {
    expect(formatDuration(0)).toBe('0s')
  })
})
