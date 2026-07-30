'use client'

import { useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError, verifyEmail } from '@/lib/api'

type State = 'loading' | 'ok' | 'error'

function VerifyInner() {
  const router = useRouter()
  const token = useSearchParams().get('token') ?? ''
  const [state, setState] = useState<State>('loading')
  const [msg, setMsg] = useState('')
  const ran = useRef(false) // strict-mode: cegah double POST

  useEffect(() => {
    if (ran.current) return
    ran.current = true
    if (!token) {
      setState('error')
      setMsg('Tautan tidak memuat token.')
      return
    }
    verifyEmail(token)
      .then(() => setState('ok'))
      .catch((e) => {
        setState('error')
        setMsg(e instanceof ApiError ? e.message : 'verifikasi gagal')
      })
  }, [token])

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle>
          {state === 'loading' && 'Memverifikasi…'}
          {state === 'ok' && 'Email terverifikasi'}
          {state === 'error' && 'Verifikasi gagal'}
        </CardTitle>
        <CardDescription>
          {state === 'loading' && 'Mohon tunggu sebentar.'}
          {state === 'ok' && 'Akun Anda aktif. Silakan masuk.'}
          {state === 'error' && msg}
        </CardDescription>
      </CardHeader>
      {state !== 'loading' && (
        <CardContent>
          <Button size="lg" className="w-full" onClick={() => router.replace('/login')}>
            Ke halaman masuk
          </Button>
        </CardContent>
      )}
    </Card>
  )
}

export default function VerifyEmailPage() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4">
      <Suspense fallback={null}>
        <VerifyInner />
      </Suspense>
    </main>
  )
}
