'use client'

import { useEffect, useState } from 'react'

import { getQuota, type SubStatus } from '@/lib/api'

// banner masa aktif langganan. Expired = fitur menulis (chat, upload, export) ditolak
// backend; membaca tetap jalan, jadi ini pemberitahuan, bukan penghalang navigasi.
export function SubscriptionBanner() {
  const [status, setStatus] = useState<SubStatus>('active')

  useEffect(() => {
    getQuota()
      .then((q) => setStatus(q.status))
      .catch(() => {})
  }, [])

  if (status === 'active') return null

  const expired = status === 'expired'
  return (
    <div
      className={`px-8 py-2 text-sm ${
        expired ? 'bg-destructive/10 text-destructive' : 'bg-accent/20 text-accent-foreground'
      }`}
    >
      {expired
        ? 'Masa aktif langganan berakhir. Percakapan dan dokumen lama tetap bisa dibuka, tapi chat baru, unggah, dan ekspor dinonaktifkan sampai langganan diperpanjang.'
        : 'Tagihan langganan menunggak. Semua fitur masih jalan selama masa tenggang, segera lakukan perpanjangan.'}
    </div>
  )
}
