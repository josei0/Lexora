'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { GoogleButton } from '@/components/google-button'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { ApiError, register } from '@/lib/api'

export default function RegisterPage() {
  const router = useRouter()
  const [firmaName, setFirmaName] = useState('')
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [done, setDone] = useState<{ needsVerify: boolean } | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await register(firmaName, fullName, email, password)
      setDone({ needsVerify: res.needs_verify })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal mendaftar, coba lagi')
      setLoading(false)
    }
  }

  if (done) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-background px-4">
        <Card className="w-full max-w-sm">
          <CardHeader>
            <CardTitle>Pendaftaran berhasil</CardTitle>
            <CardDescription>
              {done.needsVerify
                ? 'Kami kirim tautan verifikasi ke email Anda. Klik untuk mengaktifkan akun.'
                : 'Akun Anda sudah aktif. Silakan masuk.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button size="lg" className="w-full" onClick={() => router.replace('/login')}>
              Ke halaman masuk
            </Button>
          </CardContent>
        </Card>
      </main>
    )
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Daftar ke MindLaw</CardTitle>
          <CardDescription>Buat akun firma Anda. Gratis mulai paket Demo.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <Field id="firma" label="Nama firma" value={firmaName} onChange={setFirmaName} autoComplete="organization" />
            <Field id="name" label="Nama lengkap" value={fullName} onChange={setFullName} autoComplete="name" />
            <Field id="email" label="Email" type="email" value={email} onChange={setEmail} autoComplete="email" />
            <Field
              id="password"
              label="Kata sandi (min. 8 karakter)"
              type="password"
              value={password}
              onChange={setPassword}
              autoComplete="new-password"
            />
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" size="lg" disabled={loading} className="mt-2">
              {loading ? 'Memproses…' : 'Daftar'}
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              Sudah punya akun?{' '}
              <Link href="/login" className="font-medium text-primary hover:underline">
                Masuk
              </Link>
            </p>
          </form>
          <div className="mt-4">
            <GoogleButton onSuccess={() => router.replace('/app')} onError={setError} />
          </div>
        </CardContent>
      </Card>
    </main>
  )
}

function Field({
  id,
  label,
  value,
  onChange,
  type = 'text',
  autoComplete,
}: {
  id: string
  label: string
  value: string
  onChange: (v: string) => void
  type?: string
  autoComplete?: string
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium">
        {label}
      </label>
      <Input
        id={id}
        type={type}
        autoComplete={autoComplete}
        required
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  )
}
