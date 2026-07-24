'use client'

import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'

import * as apiClient from '@/lib/api'

type Status = 'loading' | 'authed' | 'anon'

type AuthState = {
  status: Status
  mustChangePassword: boolean
  login: (email: string, password: string) => Promise<apiClient.Tokens>
  logout: () => Promise<void>
  setMustChangePassword: (v: boolean) => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>('loading')
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
      setStatus('authed')
      scheduleRefresh(tok.expires_in)
    } catch {
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
      setStatus('authed')
      scheduleRefresh(tok.expires_in)
      return tok
    },
    [scheduleRefresh],
  )

  const logout = useCallback(async () => {
    if (timer.current) clearTimeout(timer.current)
    await apiClient.logout()
    setStatus('anon')
    setMustChangePassword(false)
  }, [])

  return (
    <AuthContext.Provider
      value={{ status, mustChangePassword, login, logout, setMustChangePassword }}
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
