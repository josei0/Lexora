'use client'

import { useEffect, useState } from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  ApiError,
  createTopup,
  listInvoices,
  TOPUP_PACKAGES,
  type Invoice,
} from '@/lib/api'

const STATUS_LABEL: Record<string, string> = {
  pending: 'Menunggu',
  paid: 'Lunas',
  expired: 'Kedaluwarsa',
  void: 'Dibatalkan',
}

const STATUS_CLASS: Record<string, string> = {
  pending: 'text-yellow-600',
  paid: 'text-green-600',
  expired: 'text-muted-foreground',
  void: 'text-muted-foreground',
}

function fmt(n: number) {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })
}

function daysUntil(s: string) {
  return Math.ceil((new Date(s).getTime() - Date.now()) / 86_400_000)
}

export default function BillingPage() {
  const [invoices, setInvoices] = useState<Invoice[] | null>(null)
  const [err, setErr] = useState('')
  const [buying, setBuying] = useState('')

  useEffect(() => {
    listInvoices()
      .then(setInvoices)
      .catch(e => setErr(e instanceof ApiError ? e.message : 'gagal memuat tagihan'))
  }, [])

  async function onTopup(code: 'small' | 'large') {
    setBuying(code)
    setErr('')
    try {
      const inv = await createTopup(code)
      setInvoices(prev => (prev ? [inv, ...prev] : [inv]))
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'gagal membuat top-up')
    } finally {
      setBuying('')
    }
  }

  // banner H-7: invoice pending yang masa aktifnya < 7 hari lagi
  const dueSoon = invoices?.find(
    inv => inv.status === 'pending' && inv.type === 'subscription' && daysUntil(inv.period_start) <= 7,
  )

  if (err && !invoices) return <p className="text-destructive">{err}</p>
  if (!invoices) return <p className="text-muted-foreground">Memuat…</p>

  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-6 font-serif text-3xl">Tagihan</h1>

      {err && <p className="mb-4 text-sm text-destructive">{err}</p>}

      {dueSoon && (
        <div className="mb-6 rounded-lg border border-yellow-300 bg-yellow-50 px-4 py-3 text-sm text-yellow-800">
          Invoice sebesar <strong>{fmt(dueSoon.amount_idr)}</strong> jatuh tempo{' '}
          {fmtDate(dueSoon.period_start)} — harap selesaikan pembayaran sebelum masa aktif berakhir.
        </div>
      )}

      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-base">Top-up kuota</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          {TOPUP_PACKAGES.map(p => (
            <button
              key={p.code}
              onClick={() => onTopup(p.code)}
              disabled={!!buying}
              className="rounded-lg border px-4 py-3 text-left text-sm hover:bg-accent disabled:opacity-50"
            >
              <div className="font-medium">{p.label}</div>
              <div className="text-muted-foreground">{fmt(p.price_idr)}</div>
            </button>
          ))}
          <p className="w-full text-xs text-muted-foreground">
            Kuota bertambah setelah pembayaran dikonfirmasi. Berlaku hingga akhir bulan berjalan.
          </p>
        </CardContent>
      </Card>

      {invoices.length === 0 ? (
        <p className="text-muted-foreground">Belum ada tagihan.</p>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Riwayat tagihan</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-muted-foreground">
                  <th className="px-4 py-2 font-medium">Periode</th>
                  <th className="px-4 py-2 font-medium">Jumlah</th>
                  <th className="px-4 py-2 font-medium">Status</th>
                  <th className="px-4 py-2 font-medium">Dibuat</th>
                </tr>
              </thead>
              <tbody>
                {invoices.map(inv => (
                  <tr key={inv.id} className="border-b last:border-0">
                    <td className="px-4 py-2">
                      {inv.type === 'topup'
                        ? `Top-up${inv.package_code ? ` (${inv.package_code === 'large' ? '1 juta' : '500 ribu'} token)` : ''}`
                        : `${fmtDate(inv.period_start)} – ${fmtDate(inv.period_end)}`}
                    </td>
                    <td className="px-4 py-2">{fmt(inv.amount_idr)}</td>
                    <td className={`px-4 py-2 font-medium ${STATUS_CLASS[inv.status] ?? ''}`}>
                      {STATUS_LABEL[inv.status] ?? inv.status}
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">{fmtDate(inv.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
