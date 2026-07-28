'use client'

import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { adminLogout, adminRefresh } from '@/lib/api'

// guard admin: pakai cookie admin (bukan user). Refresh proaktif tiap ~14m.
export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const [ok, setOk] = useState(false)

  useEffect(() => {
    let timer: ReturnType<typeof setInterval>
    adminRefresh()
      .then(() => {
        setOk(true)
        timer = setInterval(() => adminRefresh().catch(() => router.replace('/admin-login')), 14 * 60 * 1000)
      })
      .catch(() => router.replace('/admin-login'))
    return () => clearInterval(timer)
  }, [router])

  if (!ok) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
        Memuat…
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="flex items-center justify-between border-b px-6 py-3">
        <span className="font-serif text-lg">Panel Admin MindLaw</span>
        <button
          className="text-sm text-muted-foreground hover:text-foreground"
          onClick={() => adminLogout().then(() => router.replace('/admin-login'))}
        >
          Keluar
        </button>
      </header>
      <main className="p-8">{children}</main>
    </div>
  )
}
