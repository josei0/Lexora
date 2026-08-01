import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

// CSP + nonce per-request. BERTAHAP: mulai Report-Only (lapor, tak blok).
// Klik-jalan seluruh app nol pelanggaran -> flip ke enforce (header di bawah).
// Set CSP_ENFORCE=1 saat siap. update9-S.
const ENFORCE = process.env.CSP_ENFORCE === '1'

// API backend (fetch/SSE). Sinkron dgn lib/api.ts BASE.
const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

export function middleware(req: NextRequest) {
  // nonce 128-bit base64 (Web Crypto, edge-safe)
  const nonce = btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(16))))

  // GIS (login Google) muat script dinamis -> strict-dynamic + domain eksplisit.
  // style 'unsafe-inline': Next + Tailwind inject inline style, tak bisa di-nonce.
  const csp = [
    `default-src 'self'`,
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic' https://accounts.google.com`,
    `style-src 'self' 'unsafe-inline'`,
    `img-src 'self' data: blob: https:`,
    `font-src 'self' data:`,
    `connect-src 'self' ${API} https://accounts.google.com https://*.vercel-insights.com`,
    `frame-src https://accounts.google.com`,
    `object-src 'none'`,
    `base-uri 'self'`,
    `form-action 'self'`,
    `frame-ancestors 'none'`,
  ].join('; ')

  // forward nonce ke Next (next/script + RSC baca x-nonce)
  const reqHeaders = new Headers(req.headers)
  reqHeaders.set('x-nonce', nonce)

  const res = NextResponse.next({ request: { headers: reqHeaders } })
  res.headers.set(
    ENFORCE ? 'Content-Security-Policy' : 'Content-Security-Policy-Report-Only',
    csp,
  )
  return res
}

// lewati aset statis + gambar (tak perlu nonce, hemat CPU)
export const config = {
  matcher: [
    {
      source: '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:png|jpg|jpeg|svg|ico|webp)).*)',
      missing: [{ type: 'header', key: 'next-router-prefetch' }],
    },
  ],
}
