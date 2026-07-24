'use client'

import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'

import * as apiClient from '@/lib/api'

type Status = 'loading' | 'authed' | 'anon'

type AuthState = {
  status: Status
  role: apiClient.Role | null
  mustChangePassword: boolean
  login: (email: string, password: string) => Promise<apiClient.Tokens>
  logout: () => Promise<void>
  setMustChangePassword: (v: boolean) => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>('loading')
  const [role, setRole] = useState<apiClient.Role | null>(null)
  const [mustChangePassword, setMustChangePassword] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // proactive refresh ~1m before access token expires
  const scheduleRefresh = useCallback((expiresIn: number) => {
    if (timer.current) clearTimeout(timer.current)
    const delay = Math.max((expiresIn - 60) * 1000, 10_000)
    timer.current = setTimeout(runRefresh, delay)
  }, [])

  const runRefresh = useCallback(async () => {
    try {
      const tok = await apiClient.refresh()
      setRole(apiClient.currentRole())
      setStatus('authed')
      scheduleRefresh(tok.expires_in)
    } catch {
      setRole(null)
      setStatus('anon')
    }
  }, [scheduleRefresh])

  useEffect(() => {
    runRefresh()
    return () => {
      if (timer.current) clearTimeout(timer.current)
    }
  }, [runRefresh])

  const login = useCallback(
    async (email: string, password: string) => {
      const tok = await apiClient.login(email, password)
      setMustChangePassword(!!tok.must_change_password)
      setRole(apiClient.currentRole())
      setStatus('authed')
      scheduleRefresh(tok.expires_in)
      return tok
    },
    [scheduleRefresh],
  )

  const logout = useCallback(async () => {
    if (timer.current) clearTimeout(timer.current)
    await apiClient.logout()
    setRole(null)
    setStatus('anon')
    setMustChangePassword(false)
  }, [])

  return (
    <AuthContext.Provider
      value={{ status, role, mustChangePassword, login, logout, setMustChangePassword }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
