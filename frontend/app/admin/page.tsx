'use client'

import { CreditCard, LogOut, ScrollText, Terminal } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ApiError,
  adminLogout,
  api,
  assignSubscription,
  getPrompt,
  listAuditLogs,
  listPlans,
  setPrompt,
  type AuditLog,
  type Plan,
} from '@/lib/api'

type Org = { id: string; name: string; slug: string }
type Tab = 'sub' | 'prompt' | 'log'

const auditLabels: Record<string, string> = {
  'login.ok': 'Login berhasil',
  'login.fail': 'Login gagal',
  logout: 'Logout',
  'password.change': 'Ganti password',
  'org.create': 'Buat organisasi',
  'member.add': 'Tambah anggota',
  'member.update': 'Ubah anggota',
  'document.upload': 'Upload dokumen',
  'subscription.assign': 'Assign langganan',
  'prompt.update': 'Ubah system prompt',
}

const nav: { id: Tab; label: string; desc: string; icon: typeof CreditCard }[] = [
  { id: 'sub', label: 'Langganan', desc: 'Assign paket ke organisasi', icon: CreditCard },
  { id: 'prompt', label: 'System Prompt', desc: 'Instruksi dasar asisten', icon: Terminal },
  { id: 'log', label: 'Log Aktivitas', desc: 'Jejak audit terbaru', icon: ScrollText },
]

export default function AdminPage() {
  const router = useRouter()
  const [tab, setTab] = useState<Tab>('sub')

  const [orgs, setOrgs] = useState<Org[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [prompt, setPromptText] = useState('')
  const [promptSaved, setPromptSaved] = useState(false)
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [error, setError] = useState('')

  const [selOrg, setSelOrg] = useState('')
  const [selPlan, setSelPlan] = useState('')
  const [seats, setSeats] = useState(1)
  const [subMsg, setSubMsg] = useState('')

  useEffect(() => {
    Promise.all([
      api<Org[]>('/organizations'),
      listPlans(),
      getPrompt('system'),
      listAuditLogs(50),
    ]).then(([o, p, pr, l]) => {
      setOrgs(o)
      setPlans(p)
      setPromptText(pr.content)
      setLogs(l)
    }).catch(e => setError(e instanceof ApiError ? e.message : 'gagal memuat data'))
  }, [])

  async function handleAssign(e: React.FormEvent) {
    e.preventDefault()
    setSubMsg('')
    try {
      const sub = await assignSubscription(selOrg, selPlan, seats)
      setSubMsg(`✓ ${sub.plan.name} · ${sub.seats} seat`)
    } catch (e) {
      setSubMsg(e instanceof ApiError ? e.message : 'gagal assign')
    }
  }

  async function handlePrompt(e: React.FormEvent) {
    e.preventDefault()
    setPromptSaved(false)
    try {
      await setPrompt('system', prompt)
      setPromptSaved(true)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'gagal simpan prompt')
    }
  }

  const active = nav.find(n => n.id === tab)!

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-64 flex-col border-r border-sidebar-border bg-sidebar p-4">
        <div className="mb-8 flex items-center gap-2 px-2 pt-2">
          <span className="font-serif text-2xl text-sidebar-foreground">MindLaw</span>
          <span className="rounded bg-primary/15 px-1.5 py-0.5 text-xs font-medium text-primary">Admin</span>
        </div>
        <nav className="flex flex-1 flex-col gap-1">
          {nav.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setTab(id)}
              className={`flex items-center gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors ${
                tab === id
                  ? 'bg-sidebar-accent font-medium text-sidebar-accent-foreground'
                  : 'text-sidebar-foreground hover:bg-sidebar-accent/60'
              }`}
            >
              <Icon className="size-4 shrink-0" />
              {label}
            </button>
          ))}
        </nav>
        <Button
          variant="ghost"
          size="lg"
          className="justify-start"
          onClick={() => adminLogout().then(() => router.replace('/admin-login'))}
        >
          <LogOut className="size-4" />
          Keluar
        </Button>
      </aside>

      <main className="flex-1">
        <header className="border-b border-border px-8 py-5">
          <h1 className="font-serif text-2xl leading-tight">{active.label}</h1>
          <p className="text-sm text-muted-foreground">{active.desc}</p>
        </header>

        <div className="mx-auto max-w-2xl px-8 py-8">
          {error && <p className="mb-4 text-sm text-destructive">{error}</p>}

          {tab === 'sub' && (
            <form onSubmit={handleAssign} className="space-y-5">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Organisasi</label>
                <select
                  className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
                  value={selOrg} onChange={e => setSelOrg(e.target.value)} required
                >
                  <option value="">Pilih organisasi…</option>
                  {orgs.map(o => <option key={o.id} value={o.id}>{o.name}</option>)}
                </select>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Paket</label>
                <select
                  className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
                  value={selPlan} onChange={e => setSelPlan(e.target.value)} required
                >
                  <option value="">Pilih plan…</option>
                  {plans.map(p => <option key={p.id} value={p.code}>{p.name} · {p.monthly_token_limit > 0 ? `${(p.monthly_token_limit / 1000).toFixed(0)}k tok/seat` : 'unlimited'}</option>)}
                </select>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Jumlah seat</label>
                <Input type="number" min={1} value={seats} onChange={e => setSeats(Number(e.target.value))} className="w-32" required />
              </div>
              {subMsg && <p className="text-sm text-primary">{subMsg}</p>}
              <Button type="submit">Assign langganan</Button>
            </form>
          )}

          {tab === 'prompt' && (
            <form onSubmit={handlePrompt} className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Instruksi ini disisipkan ke setiap percakapan asisten di seluruh organisasi.
              </p>
              <textarea
                className="w-full rounded-lg border border-input bg-background px-3 py-2 font-mono text-sm leading-relaxed"
                rows={14}
                value={prompt}
                onChange={e => { setPromptText(e.target.value); setPromptSaved(false) }}
                required
              />
              <div className="flex items-center gap-3">
                <Button type="submit">Simpan prompt</Button>
                {promptSaved && <p className="text-sm text-primary">✓ Tersimpan</p>}
              </div>
            </form>
          )}

          {tab === 'log' && (
            logs.length === 0 ? (
              <p className="text-sm text-muted-foreground">Belum ada aktivitas.</p>
            ) : (
              <div className="divide-y divide-border rounded-lg border border-border">
                {logs.map(l => (
                  <div key={l.id} className="flex items-center justify-between px-4 py-3 text-sm">
                    <span>{auditLabels[l.action] ?? l.action}</span>
                    <span className="text-xs text-muted-foreground">
                      {new Date(l.created_at).toLocaleString('id-ID')} · {l.ip || '-'}
                    </span>
                  </div>
                ))}
              </div>
            )
          )}
        </div>
      </main>
    </div>
  )
}
