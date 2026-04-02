import { useCallback, useEffect, useState } from 'react'

export type Theme = 'dark' | 'light' | 'obsidian' | 'rosepine' | 'silk'

const STORAGE_KEY = 'ttyweb-theme'

const validThemes: Theme[] = ['dark', 'light', 'obsidian', 'rosepine', 'silk']

function readStoredTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored && validThemes.includes(stored as Theme)) {
      return stored as Theme
    }
  } catch {
    // localStorage unavailable
  }
  return 'dark'
}

function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute('data-theme', theme)
}

export function useTheme(): { theme: Theme; setTheme: (t: Theme) => void; toggleTheme: () => void } {
  const [theme, setThemeState] = useState<Theme>(readStoredTheme)

  useEffect(() => {
    applyTheme(theme)
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      // localStorage unavailable
    }
  }, [theme])

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t)
  }, [])

  const toggleTheme = useCallback(() => {
    setThemeState((prev) => (prev === 'dark' ? 'light' : 'dark'))
  }, [])

  return { theme, setTheme, toggleTheme }
}
