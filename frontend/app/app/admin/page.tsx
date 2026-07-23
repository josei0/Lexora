'use client'

import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  ApiError,
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

// action code → label bahasa
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

export default function AdminPage() {
  const [orgs, setOrgs] = useState<Org[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [prompt, setPromptText] = useState('')
  const [promptSaved, setPromptSaved] = useState(false)
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [error, setError] = useState('')

  // subscription form
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

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <h1 className="font-serif text-3xl">Admin</h1>
      {error && <p className="text-sm text-destructive">{error}</p>}

      <Card>
        <CardHeader><CardTitle className="text-base">Assign Langganan</CardTitle></CardHeader>
        <CardContent>
          <form onSubmit={handleAssign} className="space-y-3">
            <select
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={selOrg} onChange={e => setSelOrg(e.target.value)} required
            >
              <option value="">Pilih organisasi…</option>
              {orgs.map(o => <option key={o.id} value={o.id}>{o.name}</option>)}
            </select>
            <select
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              value={selPlan} onChange={e => setSelPlan(e.target.value)} required
            >
              <option value="">Pilih plan…</option>
              {plans.map(p => <option key={p.id} value={p.code}>{p.name} · {p.monthly_token_limit > 0 ? `${(p.monthly_token_limit / 1000).toFixed(0)}k tok/seat` : 'unlimited'}</option>)}
            </select>
            <div className="flex items-center gap-2">
              <label className="text-sm text-muted-foreground w-16">Seat</label>
              <Input type="number" min={1} value={seats} onChange={e => setSeats(Number(e.target.value))} className="w-24" required />
            </div>
            {subMsg && <p className="text-xs text-primary">{subMsg}</p>}
            <Button type="submit">Assign</Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">System Prompt</CardTitle></CardHeader>
        <CardContent>
          <form onSubmit={handlePrompt} className="space-y-3">
            <textarea
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
              rows={10}
              value={prompt}
              onChange={e => { setPromptText(e.target.value); setPromptSaved(false) }}
              required
            />
            {promptSaved && <p className="text-xs text-primary">✓ Tersimpan</p>}
            <Button type="submit">Simpan prompt</Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="text-base">Log Aktivitas</CardTitle></CardHeader>
        <CardContent>
          {logs.length === 0 ? (
            <p className="text-sm text-muted-foreground">Belum ada aktivitas.</p>
          ) : (
            <div className="divide-y text-sm">
              {logs.map(l => (
                <div key={l.id} className="flex items-center justify-between py-2">
                  <span>{auditLabels[l.action] ?? l.action}</span>
                  <span className="text-xs text-muted-foreground">
                    {new Date(l.created_at).toLocaleString('id-ID')} · {l.ip || '-'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
