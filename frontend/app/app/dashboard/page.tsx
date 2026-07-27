'use client'

import { useEffect, useState } from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError, getDashboard, getQuota, tierLabel, type DashboardStats, type Quota } from '@/lib/api'

function fmt(n: number) {
  return n.toLocaleString('id-ID')
}

function pct(used: number, limit: number) {
  if (limit === 0) return 0
  return Math.min(100, Math.round((used / limit) * 100))
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [quota, setQuota] = useState<Quota | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    Promise.all([getDashboard(), getQuota()])
      .then(([s, q]) => { setStats(s); setQuota(q) })
      .catch(e => setError(e instanceof ApiError ? e.message : 'gagal memuat dashboard'))
  }, [])

  if (error) return <p className="text-destructive">{error}</p>
  if (!stats) return <p className="text-muted-foreground">Memuat…</p>

  const used = quota?.used ?? 0
  const limit = quota?.limit ?? 0
  const p = pct(used, limit)

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-6 font-serif text-3xl">Dashboard</h1>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard title="Kualitas AI" value={tierLabel(quota?.tier)} />
        <StatCard title="Chat hari ini" value={fmt(stats.chats_today)} />
        <StatCard title="Dokumen terindeks" value={`${fmt(stats.docs_indexed)} / ${fmt(stats.docs_total)}`} />
        <StatCard title="Anggota" value={`${fmt(stats.members)}${stats.seats > 0 ? ` / ${fmt(stats.seats)} seat` : ''}`} />
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Pemakaian bulan ini</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="mb-2 flex justify-between text-sm">
            <span>{limit > 0 ? `${p}% terpakai` : 'Tanpa batas'}</span>
            <span className="text-muted-foreground">{fmt(used)} token</span>
          </div>
          {limit > 0 && (
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className={`h-full rounded-full transition-all ${p >= 100 ? 'bg-destructive' : p >= 80 ? 'bg-accent' : 'bg-primary'}`}
                style={{ width: `${p}%` }}
              />
            </div>
          )}
          {quota?.soft && !quota.hard && (
            <p className="mt-2 text-xs text-accent-foreground">Pemakaian mendekati batas (≥80%).</p>
          )}
          {quota?.hard && quota.tier === 'high' && (
            <p className="mt-2 text-xs text-accent-foreground">
              Jatah AI High bulan ini habis. Percakapan tetap jalan memakai AI Normal.
            </p>
          )}
          {quota?.hard && quota.tier === 'normal' && (
            <p className="mt-2 text-xs text-destructive">
              Jatah bulan ini habis. Hubungi admin untuk menaikkan paket.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function StatCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardContent className="pt-5">
        <p className="text-xs text-muted-foreground">{title}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  )
}
