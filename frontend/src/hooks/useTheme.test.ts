import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useTheme } from './useTheme'

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
    vi.restoreAllMocks()
  })

  it('defaults to dark theme', () => {
    const { result } = renderHook(() => useTheme())

    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
  })

  it('restores theme from localStorage', () => {
    localStorage.setItem('ttyweb-theme', 'light')

    const { result } = renderHook(() => useTheme())

    expect(result.current.theme).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })

  it('toggles from dark to light', () => {
    const { result } = renderHook(() => useTheme())

    act(() => {
      result.current.toggleTheme()
    })

    expect(result.current.theme).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    expect(localStorage.getItem('ttyweb-theme')).toBe('light')
  })

  it('toggles from light to dark', () => {
    localStorage.setItem('ttyweb-theme', 'light')
    const { result } = renderHook(() => useTheme())

    act(() => {
      result.current.toggleTheme()
    })

    expect(result.current.theme).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('ttyweb-theme')).toBe('dark')
  })

  it('handles corrupt localStorage value', () => {
    localStorage.setItem('ttyweb-theme', 'invalid')

    const { result } = renderHook(() => useTheme())

    expect(result.current.theme).toBe('dark')
  })
})
