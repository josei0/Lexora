'use client'

import { useEffect, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  ApiError,
  addMember,
  currentUserId,
  getSubscription,
  listMembers,
  updateMember,
  type Member,
  type NewMember,
  type SubscriptionView,
} from '@/lib/api'

export default function MembersPage() {
  const [members, setMembers] = useState<Member[]>([])
  const [sub, setSub] = useState<SubscriptionView | null>(null)
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [role, setRole] = useState<Member['role']>('member')
  const [created, setCreated] = useState<NewMember | null>(null)
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const me = currentUserId()
  const activeCount = members.filter((m) => m.is_active).length

  function load() {
    listMembers().then(setMembers).catch(() => {})
    getSubscription().then(setSub).catch(() => {})
  }
  useEffect(load, [])

  async function onAdd(e: React.FormEvent) {
    e.preventDefault()
    setErr('')
    setCreated(null)
    setBusy(true)
    try {
      const nm = await addMember(email, name, role)
      setCreated(nm)
      setEmail('')
      setName('')
      setRole('member')
      load()
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'gagal menambah anggota')
    } finally {
      setBusy(false)
    }
  }

  async function patch(userId: string, p: { role?: Member['role']; is_active?: boolean }) {
    setErr('')
    try {
      const updated = await updateMember(userId, p)
      setMembers((ms) => ms.map((m) => (m.user_id === userId ? updated : m)))
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'gagal memperbarui anggota')
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div className="flex items-baseline justify-between">
        <h1 className="font-serif text-3xl">Anggota</h1>
        {sub && (
          <p className="text-sm text-muted-foreground">
            {activeCount} dari {sub.seats} seat terpakai
          </p>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Tambah anggota</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onAdd} className="flex flex-wrap items-end gap-3">
            <div className="min-w-48 flex-1">
              <Input
                type="email"
                placeholder="Email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="min-w-48 flex-1">
              <Input
                placeholder="Nama lengkap"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as Member['role'])}
              className="h-9 rounded-md border border-input bg-transparent px-3 text-sm"
            >
              <option value="member">Member</option>
              <option value="org_admin">Admin</option>
            </select>
            <Button type="submit" disabled={busy}>
              {busy ? 'Menambahkan…' : 'Tambah'}
            </Button>
          </form>
          {err && <p className="mt-3 text-xs text-destructive">{err}</p>}
          {created && (
            <div className="mt-4 rounded-md border border-border bg-muted/40 p-3 text-sm">
              <p className="mb-1">
                Anggota dibuat. Berikan kata sandi sementara ini — hanya tampil sekali, wajib diganti
                saat login pertama:
              </p>
              <code className="font-mono text-sm">{created.temp_password}</code>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Daftar anggota</CardTitle>
        </CardHeader>
        <CardContent className="divide-y divide-border">
          {members.map((m) => (
            <div key={m.user_id} className="flex flex-wrap items-center gap-3 py-3">
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                  {m.full_name}
                  {m.user_id === me && <span className="ml-2 text-xs text-muted-foreground">(Anda)</span>}
                  {!m.is_active && (
                    <span className="ml-2 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                      nonaktif
                    </span>
                  )}
                </p>
                <p className="truncate text-xs text-muted-foreground">{m.email}</p>
              </div>
              <select
                value={m.role}
                disabled={m.user_id === me}
                onChange={(e) => patch(m.user_id, { role: e.target.value as Member['role'] })}
                className="h-8 rounded-md border border-input bg-transparent px-2 text-xs"
              >
                <option value="member">Member</option>
                <option value="org_admin">Admin</option>
              </select>
              <Button
                variant="ghost"
                size="sm"
                disabled={m.user_id === me}
                onClick={() => patch(m.user_id, { is_active: !m.is_active })}
              >
                {m.is_active ? 'Nonaktifkan' : 'Aktifkan'}
              </Button>
            </div>
          ))}
          {members.length === 0 && (
            <p className="py-3 text-sm text-muted-foreground">Belum ada anggota.</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
