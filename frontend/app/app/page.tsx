'use client'

import Link from 'next/link'

import { PageHeader } from '@/components/app/page-header'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuth } from '@/lib/auth-context'

export default function AppHome() {
  const { mustChangePassword } = useAuth()
  return (
    <div className="mx-auto max-w-3xl">
      <PageHeader
        title="Selamat datang di MindLaw"
        description="Platform kecerdasan hukum untuk profesional Indonesia."
      />

      {mustChangePassword && (
        <Card className="mb-6 border-accent">
          <CardHeader>
            <CardTitle className="text-lg">Ganti kata sandi sementara</CardTitle>
            <CardDescription>
              Akun Anda memakai kata sandi sementara.{' '}
              <Link href="/app/change-password" className="text-primary underline">
                Ganti sekarang
              </Link>
              .
            </CardDescription>
          </CardHeader>
        </Card>
      )}

      <Card>
        <CardContent className="pt-6 text-sm text-muted-foreground">
          Fitur riset, analisis kontrak, dan pustaka dokumen menyusul pada fase berikutnya.
        </CardContent>
      </Card>
    </div>
  )
}
