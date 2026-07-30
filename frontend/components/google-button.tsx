'use client'

import { useEffect, useRef, useState } from 'react'

import { ApiError, loginGoogle } from '@/lib/api'

// GIS (Google Identity Services) native - bukan library baru. Muat script sekali.
// Kosongnya NEXT_PUBLIC_GOOGLE_CLIENT_ID = tombol tak dirender (fitur mati).
const CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID ?? ''
const GIS_SRC = 'https://accounts.google.com/gsi/client'

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (cfg: { client_id: string; callback: (r: { credential: string }) => void }) => void
          renderButton: (el: HTMLElement, opts: Record<string, unknown>) => void
        }
      }
    }
  }
}

function loadGis(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (window.google?.accounts?.id) return resolve()
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${GIS_SRC}"]`)
    if (existing) {
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', () => reject(new Error('gagal muat GIS')))
      return
    }
    const s = document.createElement('script')
    s.src = GIS_SRC
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('gagal muat GIS'))
    document.head.appendChild(s)
  })
}

// onError: laporkan ke induk (mis. tampilkan pesan 409 "pakai login password").
// onSuccess: dipanggil setelah token diset (induk redirect).
export function GoogleButton({
  onSuccess,
  onError,
}: {
  onSuccess: () => void
  onError: (msg: string) => void
}) {
  const ref = useRef<HTMLDivElement>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (!CLIENT_ID) return
    let cancelled = false
    loadGis()
      .then(() => {
        if (cancelled || !ref.current || !window.google) return
        window.google.accounts.id.initialize({
          client_id: CLIENT_ID,
          callback: async (resp) => {
            try {
              await loginGoogle(resp.credential)
              onSuccess()
            } catch (e) {
              onError(e instanceof ApiError ? e.message : 'login Google gagal')
            }
          },
        })
        window.google.accounts.id.renderButton(ref.current, {
          theme: 'outline',
          size: 'large',
          width: 320,
          text: 'continue_with',
        })
        setReady(true)
      })
      .catch(() => onError('gagal memuat Google'))
    return () => {
      cancelled = true
    }
  }, [onSuccess, onError])

  if (!CLIENT_ID) return null

  return (
    <div className="flex flex-col items-center gap-3">
      <div className="flex w-full items-center gap-3">
        <span className="h-px flex-1 bg-border" />
        <span className="text-xs text-muted-foreground">atau</span>
        <span className="h-px flex-1 bg-border" />
      </div>
      <div ref={ref} className="min-h-[40px]">
        {!ready && <span className="text-sm text-muted-foreground">Memuat…</span>}
      </div>
    </div>
  )
}
