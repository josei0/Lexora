'use client'

import { useEffect, useState } from 'react'

import { PageHeader } from '@/components/app/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  ApiError,
  ingestWebSource,
  listWebCandidates,
  previewWebSource,
  searchWebSources,
  type WebCandidate,
  type WebPreview,
  type WebResult,
} from '@/lib/api'

function domainOf(url: string) {
  try {
    return new URL(url).hostname
  } catch {
    return url
  }
}

export default function WebSourcesPage() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<WebResult[]>([])
  const [searched, setSearched] = useState(false)
  const [preview, setPreview] = useState<(WebPreview & { url: string }) | null>(null)
  const [added, setAdded] = useState<string[]>([])
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState('')
  const [candidates, setCandidates] = useState<WebCandidate[]>([])

  useEffect(() => {
    listWebCandidates()
      .then(setCandidates)
      .catch(() => {}) // kandidat kosong bukan error yang perlu ditampilkan
  }, [])

  async function onSearch(e: React.FormEvent) {
    e.preventDefault()
    setErr('')
    setPreview(null)
    setBusy('search')
    try {
      setResults(await searchWebSources(query))
      setSearched(true)
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'pencarian gagal')
    } finally {
      setBusy('')
    }
  }

  async function onPreview(url: string) {
    setErr('')
    setBusy(url)
    try {
      const p = await previewWebSource(url)
      setPreview({ ...p, url })
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'gagal mengambil halaman')
    } finally {
      setBusy('')
    }
  }

  async function onIngest(url: string) {
    setErr('')
    setBusy(url)
    try {
      await ingestWebSource(url)
      setAdded((a) => [...a, url])
      setCandidates((c) => c.filter((x) => x.url !== url))
      setPreview(null)
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : 'gagal menambahkan ke pustaka')
    } finally {
      setBusy('')
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader
        title="Cari sumber web"
        description="Cari peraturan di situs hukum resmi, tinjau isinya, lalu tambahkan ke pustaka agar bisa dirujuk saat menjawab."
      />

      <form onSubmit={onSearch} className="flex gap-3">
        <Input
          placeholder="mis. UU 37 2004 PKPU pasal 222"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          required
        />
        <Button type="submit" disabled={busy === 'search'}>
          {busy === 'search' ? 'Mencari…' : 'Cari'}
        </Button>
      </form>

      {err && <p className="text-sm text-destructive">{err}</p>}

      {results.map((r) => {
        const isAdded = added.includes(r.url)
        return (
          <Card key={r.url}>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">{r.title || domainOf(r.url)}</CardTitle>
              <p className="text-xs text-muted-foreground">{domainOf(r.url)}</p>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="line-clamp-3 text-sm text-muted-foreground">{r.snippet}</p>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" disabled={busy === r.url} onClick={() => onPreview(r.url)}>
                  Pratinjau
                </Button>
                <Button
                  size="sm"
                  disabled={isAdded || busy === r.url || preview?.url !== r.url}
                  onClick={() => onIngest(r.url)}
                  title={preview?.url !== r.url ? 'Tinjau isinya dulu sebelum menambahkan' : undefined}
                >
                  {isAdded ? 'Sudah di pustaka' : 'Tambah ke pustaka'}
                </Button>
              </div>

              {preview?.url === r.url && (
                <div className="rounded-md border border-border bg-muted/40 p-3">
                  <p className="mb-2 text-xs text-muted-foreground">
                    {preview.title} · {preview.chars.toLocaleString('id-ID')} karakter. Teks di bawah
                    ini persis yang akan masuk pustaka.
                  </p>
                  <pre className="max-h-64 overflow-auto whitespace-pre-wrap text-xs">
                    {preview.text.slice(0, 4000)}
                  </pre>
                </div>
              )}
            </CardContent>
          </Card>
        )
      })}

      {searched && results.length === 0 && !err && (
        <p className="text-sm text-muted-foreground">
          Tidak ada hasil dari situs hukum yang diizinkan. Coba kata kunci lain.
        </p>
      )}

      {candidates.length > 0 && (
        <div className="space-y-3">
          <h2 className="font-serif text-xl">Sering dicari, belum di pustaka</h2>
          {candidates.map((c) => {
            const isAdded = added.includes(c.url)
            return (
              <Card key={c.url}>
                <CardContent className="flex items-center justify-between gap-3 py-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{domainOf(c.url)}</p>
                    <p className="truncate text-xs text-muted-foreground">{c.url}</p>
                    <p className="text-xs text-muted-foreground">{c.hits}× dicari · terakhir {c.last_at}</p>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button variant="ghost" size="sm" disabled={busy === c.url} onClick={() => onPreview(c.url)}>
                      Pratinjau
                    </Button>
                    <Button
                      size="sm"
                      disabled={isAdded || busy === c.url || preview?.url !== c.url}
                      onClick={() => onIngest(c.url)}
                      title={preview?.url !== c.url ? 'Tinjau isinya dulu sebelum menambahkan' : undefined}
                    >
                      {isAdded ? 'Sudah di pustaka' : 'Tambah'}
                    </Button>
                  </div>
                </CardContent>
                {preview?.url === c.url && (
                  <CardContent className="pt-0">
                    <div className="rounded-md border border-border bg-muted/40 p-3">
                      <p className="mb-2 text-xs text-muted-foreground">
                        {preview.title} · {preview.chars.toLocaleString('id-ID')} karakter
                      </p>
                      <pre className="max-h-64 overflow-auto whitespace-pre-wrap text-xs">
                        {preview.text.slice(0, 4000)}
                      </pre>
                    </div>
                  </CardContent>
                )}
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
