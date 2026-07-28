'use client'

import { FileText, Globe, LayoutDashboard, LogOut, MessageSquare, Receipt, Settings, ShieldCheck, Users } from 'lucide-react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'

import { useAuth } from '@/lib/auth-context'

const tenantNav = [
  { href: '/app', label: 'Beranda', icon: LayoutDashboard },
  { href: '/app/dashboard', label: 'Dashboard', icon: ShieldCheck },
  { href: '/app/chat', label: 'Chat', icon: MessageSquare },
  { href: '/app/knowledge-base', label: 'Pustaka', icon: FileText },
  { href: '/app/settings', label: 'Pengaturan', icon: Settings },
]

// disisipkan sebelum Pengaturan, org_admin only (pengelola pustaka + anggota)
const orgAdminNav = [
  { href: '/app/web-sources', label: 'Sumber web', icon: Globe },
  { href: '/app/members', label: 'Anggota', icon: Users },
  { href: '/app/billing', label: 'Tagihan', icon: Receipt },
]

// super_admin tak punya org: menu tenant pasti "akses ditolak". Tampilkan panel admin saja.
const adminNav = [{ href: '/admin', label: 'Admin', icon: ShieldCheck }]

function roleLabel(role: { system: string; org: string } | null): string {
  if (role?.system === 'super_admin') return 'Super Admin'
  if (role?.org === 'org_admin') return 'Admin Organisasi'
  return 'Anggota'
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return (parts[0][0] + (parts[1]?.[0] ?? '')).toUpperCase()
}

export function AppSidebar() {
  const { logout, role, user } = useAuth()
  const pathname = usePathname()
  const nav =
    role?.system === 'super_admin'
      ? adminNav
      : role?.org === 'org_admin'
        ? [...tenantNav.slice(0, -1), ...orgAdminNav, tenantNav[tenantNav.length - 1]]
        : tenantNav
  const name = user?.name || 'Pengguna'

  return (
    <aside className="flex w-60 flex-col border-r border-sidebar-border bg-sidebar p-4">
      <div className="mb-8 px-2 pt-1 font-serif text-2xl text-sidebar-foreground">
        MindLaw
        {role?.system === 'super_admin' && (
          <span className="ml-2 rounded bg-primary/15 px-1.5 py-0.5 align-middle text-xs font-medium text-primary">
            Admin
          </span>
        )}
      </div>

      <nav className="flex flex-1 flex-col gap-0.5">
        {nav.map(({ href, label, icon: Icon }) => {
          // '/app' exact (biar ga selalu aktif); sisanya prefix match
          const active = href === '/app' ? pathname === '/app' : pathname.startsWith(href)
          return (
            <Link
              key={href}
              href={href}
              className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                active
                  ? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground'
                  : 'text-sidebar-foreground hover:bg-sidebar-accent/60'
              }`}
            >
              <Icon className="size-4 shrink-0" />
              {label}
            </Link>
          )
        })}
      </nav>

      {/* footer profil ala Claude: avatar inisial + nama + peran, tombol keluar */}
      <div className="mt-2 flex items-center gap-2.5 border-t border-sidebar-border pt-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-primary/15 text-sm font-medium text-primary">
          {initials(name)}
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-sidebar-foreground">{name}</p>
          <p className="truncate text-xs text-muted-foreground">{roleLabel(role)}</p>
        </div>
        <button
          onClick={logout}
          aria-label="Keluar"
          title="Keluar"
          className="shrink-0 rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground"
        >
          <LogOut className="size-4" />
        </button>
      </div>
    </aside>
  )
}
