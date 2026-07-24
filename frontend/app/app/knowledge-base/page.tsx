'use client'

import { useCallback, useEffect, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { ApiError, listDocuments, uploadDocument, type Document } from '@/lib/api'
import { useAuth } from '@/lib/auth-context'

const statusLabel: Record<Document['status'], string> = {
  uploaded: 'Menunggu',
  processing: 'Memproses',
  indexed: 'Terindeks',
  failed: 'Gagal',
}

const statusClass: Record<Document['status'], string> = {
  uploaded: 'bg-muted text-muted-foreground',
  processing: 'bg-accent/20 text-accent-foreground',
  indexed: 'bg-primary/10 text-primary',
  failed: 'bg-destructive/10 text-destructive',
}

export default function KnowledgeBasePage() {
  const { role } = useAuth()
  const canUpload = role?.org === 'org_admin'
  const [docs, setDocs] = useState<Document[]>([])
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const load = useCallback(async () => {
    try {
      setDocs(await listDocuments())
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal memuat dokumen')
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  // poll while anything is still processing
  useEffect(() => {
    const pending = docs.some((d) => d.status === 'uploaded' || d.status === 'processing')
    if (!pending) return
    const t = setInterval(load, 3000)
    return () => clearInterval(t)
  }, [docs, load])

  async function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setError('')
    setUploading(true)
    try {
      await uploadDocument(file)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal mengunggah')
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="font-serif text-3xl">Pustaka Pengetahuan</h1>
          <p className="text-muted-foreground">
            {canUpload
              ? 'Unggah dokumen hukum (PDF, DOCX, TXT, maks 20MB). PDF harus versi teks; hasil scan belum didukung.'
              : 'Dokumen di pustaka pengetahuan organisasi Anda. Hanya admin organisasi yang bisa mengunggah.'}
          </p>
        </div>
        {canUpload && (
          <div>
            <input
              ref={inputRef}
              type="file"
              accept=".pdf,.docx,.txt"
              onChange={onFile}
              className="hidden"
              id="doc-upload"
            />
            <Button size="lg" disabled={uploading} onClick={() => inputRef.current?.click()}>
              {uploading ? 'Mengunggah…' : 'Unggah dokumen'}
            </Button>
          </div>
        )}
      </div>

      {error && <p className="mb-4 text-sm text-destructive">{error}</p>}

      {docs.length === 0 ? (
        <Card>
          <CardContent className="pt-6 text-sm text-muted-foreground">
            Belum ada dokumen. Unggah file pertama untuk mulai mengindeks.
          </CardContent>
        </Card>
      ) : (
        <div className="flex flex-col gap-2">
          {docs.map((d) => (
            <Card key={d.id}>
              <CardContent className="flex items-center justify-between py-4">
                <div className="min-w-0">
                  <p className="truncate font-medium">{d.file_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {(d.file_size / 1024).toFixed(0)} KB
                    {d.status === 'failed' && d.error ? ` · ${d.error}` : ''}
                  </p>
                </div>
                <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${statusClass[d.status]}`}>
                  {statusLabel[d.status]}
                </span>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
