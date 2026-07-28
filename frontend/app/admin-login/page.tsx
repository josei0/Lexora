'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { adminEnroll, adminLogin, adminVerify, ApiError } from '@/lib/api'

type Step = 'credentials' | 'enroll' | 'verify' | 'recovery'

// ponytail: secret manual entry, QR image kalau dibutuhkan (butuh lib qrcode)
function secretFrom(otpauthUrl: string): string {
  try {
    return new URL(otpauthUrl).searchParams.get('secret') ?? ''
  } catch {
    return ''
  }
}

export default function AdminLoginPage() {
  const router = useRouter()
  const [step, setStep] = useState<Step>('credentials')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [secret, setSecret] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  function fail(err: unknown) {
    setError(err instanceof ApiError ? err.message : 'gagal, coba lagi')
    setLoading(false)
  }

  async function onCredentials(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await adminLogin(email, password)
      if (res.enroll_required && res.otpauth_url) {
        setSecret(secretFrom(res.otpauth_url))
        setStep('enroll')
      } else {
        setStep('verify')
      }
      setLoading(false)
    } catch (err) {
      fail(err)
    }
  }

  async function onCode(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      if (step === 'enroll') {
        const tok = await adminEnroll(email, password, code)
        setRecoveryCodes(tok.recovery_codes ?? [])
        setStep('recovery')
        setLoading(false)
      } else {
        await adminVerify(email, password, code)
        router.replace('/admin')
      }
    } catch (err) {
      fail(err)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>Panel Admin MindLaw</CardTitle>
          <CardDescription>
            {step === 'credentials' && 'Khusus super admin. Verifikasi dua langkah wajib.'}
            {step === 'enroll' && 'Aktifkan aplikasi authenticator untuk akun ini.'}
            {step === 'verify' && 'Masukkan kode dari aplikasi authenticator.'}
            {step === 'recovery' && 'Simpan kode pemulihan ini. Hanya tampil sekali.'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {step === 'credentials' && (
            <form onSubmit={onCredentials} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <label htmlFor="email" className="text-sm font-medium">
                  Email
                </label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <label htmlFor="password" className="text-sm font-medium">
                  Kata sandi
                </label>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
              {error && <p className="text-sm text-destructive">{error}</p>}
              <Button type="submit" size="lg" disabled={loading} className="mt-2">
                {loading ? 'Memproses…' : 'Lanjut'}
              </Button>
            </form>
          )}

          {(step === 'enroll' || step === 'verify') && (
            <form onSubmit={onCode} className="flex flex-col gap-4">
              {step === 'enroll' && (
                <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
                  <p className="mb-2">
                    Tambahkan akun di aplikasi authenticator (Google Authenticator, Aegis, 1Password)
                    lewat entri manual dengan secret berikut:
                  </p>
                  <code className="block break-all rounded bg-background p-2 font-mono text-xs">{secret}</code>
                </div>
              )}
              <div className="flex flex-col gap-1.5">
                <label htmlFor="code" className="text-sm font-medium">
                  {step === 'enroll' ? 'Kode 6 digit dari aplikasi' : 'Kode 6 digit atau kode pemulihan'}
                </label>
                <Input
                  id="code"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  required
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                />
              </div>
              {error && <p className="text-sm text-destructive">{error}</p>}
              <Button type="submit" size="lg" disabled={loading} className="mt-2">
                {loading ? 'Memproses…' : 'Verifikasi'}
              </Button>
            </form>
          )}

          {step === 'recovery' && (
            <div className="flex flex-col gap-4">
              <ul className="grid grid-cols-2 gap-2 rounded-md border border-border bg-muted/40 p-3 font-mono text-sm">
                {recoveryCodes.map((c) => (
                  <li key={c}>{c}</li>
                ))}
              </ul>
              <p className="text-sm text-muted-foreground">
                Tiap kode hanya bisa dipakai sekali, untuk masuk saat aplikasi authenticator hilang.
              </p>
              <Button size="lg" onClick={() => router.replace('/admin')}>
                Lanjut ke panel
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </main>
  )
}
