import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'

const ThemeContext = createContext(null)

const STORAGE_KEY = 'pulsepoll_theme'

/**
 * Gets the effective theme to use:
 * 1. User's explicit stored choice takes priority
 * 2. Fall back to OS preference
 * 3. Ultimate fallback: dark
 */
function getInitialTheme() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'dark' || stored === 'light') return stored
  } catch (_) {
    // localStorage not available (e.g. private mode with restrictions)
  }
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return 'dark'
}

function applyTheme(theme) {
  const root = document.documentElement
  root.setAttribute('data-theme', theme)
  root.style.colorScheme = theme
}

export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(getInitialTheme)
  // Track whether user manually set the preference
  const [userOverride, setUserOverride] = useState(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      return stored === 'dark' || stored === 'light'
    } catch (_) {
      return false
    }
  })

  // Apply theme to <html> on every change
  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  // Listen for OS preference changes — only if user has NOT set a manual override
  useEffect(() => {
    if (userOverride) return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = (e) => {
      setTheme(e.matches ? 'dark' : 'light')
    }
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [userOverride])

  const toggleTheme = useCallback(() => {
    setTheme(prev => {
      const next = prev === 'dark' ? 'light' : 'dark'
      try {
        localStorage.setItem(STORAGE_KEY, next)
      } catch (_) {}
      setUserOverride(true)
      return next
    })
  }, [])

  return (
    <ThemeContext.Provider value={{ theme, isDark: theme === 'dark', toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
