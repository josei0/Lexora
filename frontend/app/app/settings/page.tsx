'use client'

import { useEffect, useState } from 'react'

import { PageHeader } from '@/components/app/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { ApiError, api, getSubscription, type SubscriptionView } from '@/lib/api'

export default function SettingsPage() {
  const [sub, setSub] = useState<SubscriptionView | null>(null)
  const [subLoaded, setSubLoaded] = useState(false)
  const [cur, setCur] = useState('')
  const [next, setNext] = useState('')
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    getSubscription().then(setSub).catch(() => {})
  }, [])

  async function changePassword(e: React.FormEvent) {
    e.preventDefault()
    setMsg(''); setErr('')
    setBusy(true)
    try {
      await api('/auth/change-password', {
        method: 'POST',
        body: JSON.stringify({ current_password: cur, new_password: next }),
      })
      setMsg('Kata sandi berhasil diubah.')
      setCur(''); setNext('')
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'gagal mengubah kata sandi')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <PageHeader title="Pengaturan" description="Kelola akun dan keamanan Anda." />

      <Card>
        <CardHeader><CardTitle className="text-base">Ganti kata sandi</CardTitle></CardHeader>
        <CardContent>
          <form onSubmit={changePassword} className="space-y-3">
            <Input type="password" placeholder="Kata sandi saat ini" value={cur} onChange={e => setCur(e.target.value)} required />
            <Input type="password" placeholder="Kata sandi baru (min. 8 karakter)" value={next} onChange={e => setNext(e.target.value)} minLength={8} required />
            {err && <p className="text-xs text-destructive">{err}</p>}
            {msg && <p className="text-xs text-primary">{msg}</p>}
            <Button type="submit" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan'}</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
