import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [token, setToken] = useState(() => localStorage.getItem('pulsepoll_token'))
  const [loading, setLoading] = useState(true)

  const fetchUser = useCallback(async () => {
    if (!localStorage.getItem('pulsepoll_token')) {
      setUser(null)
      setLoading(false)
      return
    }

    try {
      const userData = await api.auth.me()
      setUser(userData)
    } catch (err) {
      console.warn('Failed to restore session:', err)
      localStorage.removeItem('pulsepoll_token')
      setToken(null)
      setUser(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  const login = async (email, password) => {
    const data = await api.auth.login({ email, password })
    localStorage.setItem('pulsepoll_token', data.token)
    setToken(data.token)
    setUser(data.user)
    return data
  }

  const register = async (name, email, password) => {
    const data = await api.auth.register({ name, email, password })
    localStorage.setItem('pulsepoll_token', data.token)
    setToken(data.token)
    setUser(data.user)
    return data
  }

  const logout = () => {
    localStorage.removeItem('pulsepoll_token')
    setToken(null)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, token, loading, isAuthenticated: !!user, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
