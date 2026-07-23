'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { api, ApiError } from '@/lib/api'
import { useAuth } from '@/lib/auth-context'

export default function ChangePasswordPage() {
  const router = useRouter()
  const { setMustChangePassword } = useAuth()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (next.length < 8) return setError('kata sandi baru minimal 8 karakter')
    if (next !== confirm) return setError('konfirmasi kata sandi tidak cocok')
    setLoading(true)
    try {
      await api('/auth/change-password', {
        method: 'POST',
        body: JSON.stringify({ current_password: current, new_password: next }),
      })
      setMustChangePassword(false)
      router.replace('/app')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal mengganti kata sandi')
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-sm">
      <Card>
        <CardHeader>
          <CardTitle>Ganti kata sandi</CardTitle>
          <CardDescription>Buat kata sandi baru untuk akun Anda.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="current" className="text-sm font-medium">
                Kata sandi saat ini
              </label>
              <Input
                id="current"
                type="password"
                autoComplete="current-password"
                required
                value={current}
                onChange={(e) => setCurrent(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label htmlFor="next" className="text-sm font-medium">
                Kata sandi baru
              </label>
              <Input
                id="next"
                type="password"
                autoComplete="new-password"
                required
                value={next}
                onChange={(e) => setNext(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label htmlFor="confirm" className="text-sm font-medium">
                Konfirmasi kata sandi baru
              </label>
              <Input
                id="confirm"
                type="password"
                autoComplete="new-password"
                required
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button type="submit" size="lg" disabled={loading} className="mt-2">
              {loading ? 'Menyimpan…' : 'Simpan'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
