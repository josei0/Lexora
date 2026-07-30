'use client'

import { Building2, CreditCard, LogOut, ScrollText, Terminal } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  ApiError,
  adminLogout,
  api,
  assignMemberToOrg,
  assignSubscription,
  createOrganization,
  getPrompt,
  listAuditLogs,
  listPlans,
  setPrompt,
  type AuditLog,
  type NewMember,
  type Plan,
} from '@/lib/api'

type Org = { id: string; name: string; slug: string }
type Tab = 'org' | 'sub' | 'prompt' | 'log'

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
  { id: 'org', label: 'Organisasi', desc: 'Buat organisasi atau assign akun', icon: Building2 },
  { id: 'sub', label: 'Langganan', desc: 'Assign paket ke organisasi', icon: CreditCard },
  { id: 'prompt', label: 'System Prompt', desc: 'Instruksi dasar asisten', icon: Terminal },
  { id: 'log', label: 'Log Aktivitas', desc: 'Jejak audit terbaru', icon: ScrollText },
]

export default function AdminPage() {
  const router = useRouter()
  const [tab, setTab] = useState<Tab>('org')

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

  // buat organisasi baru
  const [orgName, setOrgName] = useState('')
  const [orgAdminEmail, setOrgAdminEmail] = useState('')
  const [orgAdminName, setOrgAdminName] = useState('')
  const [createResult, setCreateResult] = useState<NewMember | null>(null)
  const [createErr, setCreateErr] = useState('')

  // assign akun ke org existing (default: mindlaw)
  const [assignOrg, setAssignOrg] = useState('')
  const [assignEmail, setAssignEmail] = useState('')
  const [assignName, setAssignName] = useState('')
  const [assignResult, setAssignResult] = useState<NewMember | null>(null)
  const [assignErr, setAssignErr] = useState('')

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
      // default assign ke org rumah "mindlaw" kalau ada
      const home = o.find(x => x.slug === 'mindlaw') ?? o[0]
      if (home) setAssignOrg(home.id)
    }).catch(e => setError(e instanceof ApiError ? e.message : 'gagal memuat data'))
  }, [])

  async function handleCreateOrg(e: React.FormEvent) {
    e.preventDefault()
    setCreateErr(''); setCreateResult(null)
    const slug = orgName.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
    try {
      const res = await createOrganization(orgName, slug, orgAdminEmail, orgAdminName)
      setCreateResult(res.admin)
      setOrgs(prev => [{ id: res.organization.id, name: res.organization.name, slug: res.organization.slug }, ...prev])
      setOrgName(''); setOrgAdminEmail(''); setOrgAdminName('')
    } catch (e) {
      setCreateErr(e instanceof ApiError ? e.message : 'gagal membuat organisasi')
    }
  }

  async function handleAssign2(e: React.FormEvent) {
    e.preventDefault()
    setAssignErr(''); setAssignResult(null)
    try {
      const nm = await assignMemberToOrg(assignOrg, assignEmail, assignName)
      setAssignResult(nm)
      setAssignEmail(''); setAssignName('')
    } catch (e) {
      setAssignErr(e instanceof ApiError ? e.message : 'gagal assign akun')
    }
  }

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

          {tab === 'org' && (
            <div className="space-y-10">
              <form onSubmit={handleCreateOrg} className="space-y-5">
                <div>
                  <h2 className="text-sm font-semibold">Buat Organisasi</h2>
                  <p className="text-sm text-muted-foreground">Bikin firma baru + akun admin pertamanya.</p>
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Nama organisasi</label>
                  <Input value={orgName} onChange={e => setOrgName(e.target.value)} required />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Nama admin</label>
                  <Input value={orgAdminName} onChange={e => setOrgAdminName(e.target.value)} required />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Email admin</label>
                  <Input type="email" value={orgAdminEmail} onChange={e => setOrgAdminEmail(e.target.value)} required />
                </div>
                {createErr && <p className="text-sm text-destructive">{createErr}</p>}
                {createResult && <CredCard nm={createResult} />}
                <Button type="submit">Buat organisasi</Button>
              </form>

              <form onSubmit={handleAssign2} className="space-y-5 border-t border-border pt-8">
                <div>
                  <h2 className="text-sm font-semibold">Assign Akun</h2>
                  <p className="text-sm text-muted-foreground">Tambah 1 akun ke organisasi yang sudah ada (mis. mindlaw).</p>
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Organisasi</label>
                  <select
                    className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
                    value={assignOrg} onChange={e => setAssignOrg(e.target.value)} required
                  >
                    {orgs.map(o => <option key={o.id} value={o.id}>{o.name}{o.slug === 'mindlaw' ? ' (internal)' : ''}</option>)}
                  </select>
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Nama</label>
                  <Input value={assignName} onChange={e => setAssignName(e.target.value)} required />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Email</label>
                  <Input type="email" value={assignEmail} onChange={e => setAssignEmail(e.target.value)} required />
                </div>
                {assignErr && <p className="text-sm text-destructive">{assignErr}</p>}
                {assignResult && <CredCard nm={assignResult} />}
                <Button type="submit">Assign akun</Button>
              </form>
            </div>
          )}

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

// kartu hasil: email + temp password sekali tampil (super_admin salin ke user)
function CredCard({ nm }: { nm: NewMember }) {
  return (
    <div className="space-y-1 rounded-lg border border-primary/30 bg-primary/5 p-4 text-sm">
      <p className="font-medium text-primary">✓ Akun dibuat</p>
      <p>Email: <span className="font-mono">{nm.email}</span></p>
      <p>Password sementara: <span className="font-mono">{nm.temp_password}</span></p>
      <p className="text-xs text-muted-foreground">Salin dan berikan ke pengguna. Hanya tampil sekali.</p>
    </div>
  )
}
