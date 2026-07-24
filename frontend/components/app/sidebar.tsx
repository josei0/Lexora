'use client'

import { FileText, LayoutDashboard, LogOut, MessageSquare, Settings, ShieldCheck } from 'lucide-react'
import Link from 'next/link'

import { Button } from '@/components/ui/button'
import { useAuth } from '@/lib/auth-context'

const tenantNav = [
  { href: '/app', label: 'Beranda', icon: LayoutDashboard },
  { href: '/app/dashboard', label: 'Dashboard', icon: ShieldCheck },
  { href: '/app/chat', label: 'Chat', icon: MessageSquare },
  { href: '/app/knowledge-base', label: 'Pustaka', icon: FileText },
  { href: '/app/settings', label: 'Pengaturan', icon: Settings },
]

// super_admin tak punya org: menu tenant pasti "akses ditolak". Tampilkan panel admin saja.
const adminNav = [{ href: '/app/admin', label: 'Admin', icon: ShieldCheck }]

export function AppSidebar() {
  const { logout, role } = useAuth()
  const nav = role?.system === 'super_admin' ? adminNav : tenantNav
  return (
    <aside className="flex w-60 flex-col border-r border-sidebar-border bg-sidebar p-4">
      <div className="mb-8 px-2 font-serif text-2xl text-sidebar-foreground">
        Lexora
        {role?.system === 'super_admin' && (
          <span className="ml-2 rounded bg-primary/15 px-1.5 py-0.5 align-middle text-xs font-medium text-primary">
            Admin
          </span>
        )}
      </div>
      <nav className="flex flex-1 flex-col gap-1">
        {nav.map(({ href, label, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm text-sidebar-foreground transition-colors hover:bg-sidebar-accent"
          >
            <Icon className="size-4" />
            {label}
          </Link>
        ))}
      </nav>
      <Button variant="ghost" size="lg" onClick={logout} className="justify-start">
        <LogOut className="size-4" />
        Keluar
      </Button>
    </aside>
  )
}
