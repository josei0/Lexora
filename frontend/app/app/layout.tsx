'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

import { AuthProvider, useAuth } from '@/lib/auth-context'
import { AppSidebar } from '@/components/app/sidebar'

function Guard({ children }: { children: React.ReactNode }) {
  const { status } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (status === 'anon') router.replace('/login')
  }, [status, router])

  if (status !== 'authed') {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
        Memuat…
      </div>
    )
  }
  return (
    <div className="flex min-h-screen bg-background">
      <AppSidebar />
      <main className="flex-1 p-8">{children}</main>
    </div>
  )
}

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <Guard>{children}</Guard>
    </AuthProvider>
  )
}
