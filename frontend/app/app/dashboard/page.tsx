'use client'

import { useEffect, useState } from 'react'

import { PageHeader } from '@/components/app/page-header'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError, getDashboard, getQuota, tierLabel, type DashboardStats, type Quota, type QuotaWindow } from '@/lib/api'

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
      <PageHeader title="Dashboard" description="Ringkasan pemakaian dan aktivitas organisasi Anda." />

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <StatCard title="Kualitas AI" value={tierLabel(quota?.tier)} />
        <StatCard title="Chat hari ini" value={fmt(stats.chats_today)} />
        <StatCard title="Dokumen terindeks" value={`${fmt(stats.docs_indexed)} / ${fmt(stats.docs_total)}`} />
        <StatCard title="Anggota" value={`${fmt(stats.members)}${stats.seats > 0 ? ` / ${fmt(stats.seats)} seat` : ''}`} />
      </div>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="text-base">Batas pemakaian</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {(quota?.windows?.filter((w) => w.limit > 0) ?? []).map((w) => (
            <WindowMeter key={w.kind} w={w} />
          ))}
          {/* fallback lama: tanpa breakdown window (org tanpa subscription / kuota bulanan saja) */}
          {!quota?.windows?.some((w) => w.limit > 0) && (
            <div>
              <div className="mb-2 flex justify-between text-sm">
                <span>{limit > 0 ? `${p}% terpakai bulan ini` : 'Tanpa batas'}</span>
                <span className="text-muted-foreground">{fmt(used)} token</span>
              </div>
              {limit > 0 && <MeterBar p={p} />}
            </div>
          )}
          {quota?.soft && !quota.hard && (
            <p className="text-xs text-accent-foreground">Pemakaian mendekati batas (≥80%).</p>
          )}
          {quota?.hard && (
            <p className="text-xs text-destructive">
              Batas tercapai — tunggu reset atau beli saldo lanjut di halaman Tagihan.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

const WINDOW_LABEL: Record<QuotaWindow['kind'], string> = {
  session: 'Sesi (5 jam)',
  weekly: 'Mingguan',
  monthly: 'Bulanan',
}

function MeterBar({ p }: { p: number }) {
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
      <div
        className={`h-full rounded-full transition-all ${p >= 100 ? 'bg-destructive' : p >= 80 ? 'bg-accent' : 'bg-primary'}`}
        style={{ width: `${p}%` }}
      />
    </div>
  )
}

// meter satu window (update8): label + % + token + kapan reset
function WindowMeter({ w }: { w: QuotaWindow }) {
  const p = pct(w.used, w.limit)
  return (
    <div>
      <div className="mb-2 flex justify-between text-sm">
        <span>
          {WINDOW_LABEL[w.kind]} — {p}% terpakai
          <span className="ml-2 text-xs text-muted-foreground">reset {resetLabel(w.kind, w.reset_at)}</span>
        </span>
        <span className="text-muted-foreground">
          {fmt(w.used)} / {fmt(w.limit)} token
        </span>
      </div>
      <MeterBar p={p} />
    </div>
  )
}

function resetLabel(kind: QuotaWindow['kind'], resetAt: string): string {
  if (kind === 'session') {
    return new Date(resetAt).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
  }
  return new Date(resetAt).toLocaleDateString('id-ID', { day: 'numeric', month: 'short' })
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
