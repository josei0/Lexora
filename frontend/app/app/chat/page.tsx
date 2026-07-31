'use client'

import { Plus, Trash2, Download, Globe, Paperclip, X } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  ApiError,
  askStream,
  chatMessages,
  createChat,
  createTopup,
  deleteChat,
  exportChat,
  listChats,
  type Chat,
  type Citation,
  type Message,
  type QuotaBlock,
} from '@/lib/api'

export default function ChatPage() {
  const [chats, setChats] = useState<Chat[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState('')
  const [citations, setCitations] = useState<Citation[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [quotaBlock, setQuotaBlock] = useState<QuotaBlock | null>(null) // paywall di titik limit (update8 F3)
  const [buying, setBuying] = useState('')
  const [quotaNotice, setQuotaNotice] = useState('')
  const [quotaLow, setQuotaLow] = useState(false)
  const [webSearch, setWebSearch] = useState(false)
  const [status, setStatus] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const bottomRef = useRef<HTMLDivElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  const loadChats = useCallback(async () => {
    try {
      setChats(await listChats())
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal memuat percakapan')
    }
  }, [])

  useEffect(() => {
    loadChats()
  }, [loadChats])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streaming])

  async function openChat(id: string) {
    setActiveId(id)
    setStreaming('')
    setCitations([])
    setError('')
    try {
      setMessages(await chatMessages(id))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal memuat pesan')
    }
  }

  function newChat() {
    setActiveId(null)
    setMessages([])
    setStreaming('')
    setCitations([])
    setError('')
  }

  async function removeChat(id: string) {
    try {
      await deleteChat(id)
      if (id === activeId) newChat()
      await loadChats()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal menghapus percakapan')
    }
  }

  async function send(e?: React.FormEvent) {
    e?.preventDefault()
    const question = input.trim()
    if ((!question && files.length === 0) || busy) return

    const sent = files
    const label = question || sent.map((f) => `📎 ${f.name}`).join(' ')
    setError('')
    setQuotaBlock(null)
    setBusy(true)
    setInput('')
    setFiles([])
    setStreaming('')
    setCitations([])
    setMessages((prev) => [
      ...prev,
      { id: `local-${Date.now()}`, role: 'user', content: label, created_at: '', citations: [] },
    ])

    try {
      let chatId = activeId
      if (!chatId) {
        const chat = await createChat(question || 'Lampiran')
        chatId = chat.id
        setActiveId(chat.id)
      }

      let answer = ''
      await askStream(chatId, question, sent, {
        onToken: (tok) => {
          answer += tok
          setStreaming(answer)
        },
        onStatus: (s) => setStatus(s === 'searching' ? 'Mencari di web…' : ''),
        onDone: (cits, messageId, quota, web) => {
          setMessages((prev) => [
            ...prev,
            { id: messageId, role: 'assistant', content: answer, created_at: '', citations: cits },
          ])
          setStreaming('')
          setStatus('')
          setCitations(cits)
          setQuotaNotice(
            web?.skipped === 'quota'
              ? 'Jatah pencarian web hari ini habis. Jawaban disusun dari pustaka.'
              : web?.skipped === 'failed'
                ? 'Pencarian web gagal. Jawaban disusun dari pustaka.'
                : web?.skipped === 'plan'
                  ? 'Pencarian web tidak tersedia di paket ini.'
                  : quota?.degraded
                    ? 'Jatah AI High bulan ini habis. Jawaban berikutnya memakai AI Normal.'
                    : quota?.soft
                      ? 'Pemakaian bulan ini sudah di atas 80%.'
                      : '',
          )
          setQuotaLow(!!(quota?.degraded || quota?.soft))
        },
        onError: (msg, block) => {
          setError(msg)
          setQuotaBlock(block ?? null) // kaya info -> banner "beli saldo lanjut"
        },
      }, webSearch)
      await loadChats()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal mengirim pertanyaan')
    } finally {
      setBusy(false)
    }
  }

  // beli saldo lanjut langsung dari banner limit (update8 F3) — alur top-up Mayar existing
  async function onBuy(code: string) {
    setBuying(code)
    setError('')
    try {
      const inv = await createTopup(code as 'small' | 'large')
      if (inv.checkout_url) {
        window.location.href = inv.checkout_url
        return
      }
      setError('Tagihan dibuat — selesaikan di halaman Tagihan.')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'gagal membuat top-up')
    } finally {
      setBuying('')
    }
  }

  return (
    <div className="flex h-[calc(100vh-6rem)] gap-6">
      <aside className="flex w-64 flex-col gap-2">
        <Button onClick={newChat} variant="outline" className="justify-start">
          <Plus className="size-4" />
          Percakapan baru
        </Button>
        <div className="flex flex-1 flex-col gap-1 overflow-y-auto">
          {chats.map((c) => (
            <div
              key={c.id}
              className={`group flex items-center gap-1 rounded-lg px-3 py-2 text-sm transition-colors hover:bg-muted ${
                c.id === activeId ? 'bg-muted' : ''
              }`}
            >
              <button onClick={() => openChat(c.id)} className="min-w-0 flex-1 truncate text-left">
                {c.title}
              </button>
              <button
                onClick={() => removeChat(c.id)}
                aria-label={`Hapus ${c.title}`}
                className="opacity-0 transition-opacity group-hover:opacity-100"
              >
                <Trash2 className="size-3.5 text-muted-foreground" />
              </button>
            </div>
          ))}
        </div>
      </aside>

      <section className="flex min-w-0 flex-1 flex-col">
        {activeId && messages.some((m) => !m.id.startsWith('local-')) && (
          <div className="mb-2 flex justify-end gap-2">
            <Button variant="ghost" size="sm" onClick={() => exportChat(activeId, 'word')}>
              <Download className="size-3.5" />
              Word
            </Button>
            <Button variant="ghost" size="sm" onClick={() => exportChat(activeId, 'pdf')}>
              <Download className="size-3.5" />
              HTML/PDF
            </Button>
          </div>
        )}
        <div className="flex-1 overflow-y-auto pr-2">
          {messages.length === 0 && !streaming ? (
            <div className="flex h-full items-center justify-center text-center">
              <div>
                <h1 className="font-serif text-3xl">Tanya MindLaw</h1>
                <p className="mt-2 text-muted-foreground">
                  Jawaban disusun dari dokumen di pustaka pengetahuan organisasi Anda.
                </p>
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              {messages.map((m) => (
                <MessageBubble key={m.id} message={m} />
              ))}
              {streaming && (
                <MessageBubble
                  message={{ id: 'streaming', role: 'assistant', content: streaming, created_at: '', citations }}
                />
              )}
            </div>
          )}
          <div ref={bottomRef} />
        </div>

        {error && <p className="py-2 text-sm text-destructive">{error}</p>}
        {quotaBlock && (
          <div className="mt-2 rounded-lg border border-yellow-300 bg-yellow-50 p-4 text-sm">
            <p className="font-medium text-yellow-900">
              Batas {WINDOW_LABEL[quotaBlock.window]} tercapai — reset {resetIn(quotaBlock.reset_at)}.
            </p>
            <p className="mt-1 text-yellow-800">
              Beli saldo lanjut untuk gas terus tanpa nunggu reset:
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              {quotaBlock.packages.map((p) => (
                <Button
                  key={p.code}
                  size="sm"
                  disabled={buying !== ''}
                  onClick={() => onBuy(p.code)}
                >
                  {buying === p.code ? 'Memproses…' : `${p.label} — ${fmtIDR(p.price_idr)}`}
                </Button>
              ))}
              <Button variant="ghost" size="sm" onClick={() => setQuotaBlock(null)}>
                Nanti saja
              </Button>
            </div>
          </div>
        )}
        {status && <p className="py-2 text-xs text-muted-foreground">{status}</p>}
        {quotaNotice && (
          <p className="py-2 text-xs text-accent-foreground">
            {quotaNotice}
            {quotaLow && (
              <>
                {' '}
                <a href="/app/billing" className="underline">
                  Top-up kuota
                </a>
              </>
            )}
          </p>
        )}

        {files.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-2">
            {files.map((f, i) => (
              <span key={`${f.name}-${i}`} className="flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs">
                <Paperclip className="size-3" />
                <span className="max-w-40 truncate">{f.name}</span>
                <button
                  type="button"
                  aria-label={`Hapus ${f.name}`}
                  onClick={() => setFiles((prev) => prev.filter((_, j) => j !== i))}
                >
                  <X className="size-3" />
                </button>
              </span>
            ))}
          </div>
        )}

        <form onSubmit={send} className="mt-4 flex items-end gap-2">
          <input
            ref={fileRef}
            type="file"
            multiple
            accept="image/*,.pdf,.docx,.txt"
            className="hidden"
            onChange={(e) => {
              setFiles((prev) => [...prev, ...Array.from(e.target.files ?? [])])
              e.target.value = ''
            }}
          />
          <Button
            type="button"
            variant="outline"
            size="icon"
            disabled={busy}
            onClick={() => fileRef.current?.click()}
            aria-label="Lampirkan file"
          >
            <Paperclip className="size-4" />
          </Button>
          <Button
            type="button"
            variant={webSearch ? 'default' : 'outline'}
            size="icon"
            // lampiran menang: toggle mati saat ada file (backend juga mengabaikannya)
            disabled={busy || files.length > 0}
            onClick={() => setWebSearch((v) => !v)}
            aria-pressed={webSearch}
            title={
              files.length > 0
                ? 'Tidak tersedia saat ada lampiran'
                : 'Cari di web. Pertanyaan Anda dikirim ke layanan pencarian.'
            }
            aria-label="Cari di web"
          >
            <Globe className="size-4" />
          </Button>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              // Enter kirim, Shift+Enter baris baru
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                send()
              }
            }}
            rows={1}
            placeholder="Tulis pertanyaan hukum… (Shift+Enter untuk baris baru)"
            disabled={busy}
            className="flex max-h-40 min-h-9 flex-1 resize-none rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50"
          />
          <Button type="submit" disabled={busy || (!input.trim() && files.length === 0)}>
            {busy ? 'Menjawab…' : 'Kirim'}
          </Button>
        </form>
      </section>
    </div>
  )
}

const WINDOW_LABEL: Record<QuotaBlock['window'], string> = {
  session: 'sesi 5 jam',
  weekly: 'mingguan',
  monthly: 'bulanan',
}

// "reset dalam 3 hari" / "reset dalam 4 jam" — tanpa lib tanggal
function resetIn(resetAt: string): string {
  const ms = new Date(resetAt).getTime() - Date.now()
  if (ms <= 0) return 'sebentar lagi'
  const jam = Math.ceil(ms / 3_600_000)
  if (jam < 24) return `dalam ${jam} jam`
  return `dalam ${Math.ceil(jam / 24)} hari`
}

function fmtIDR(n: number): string {
  return `Rp${n.toLocaleString('id-ID')}`
}

function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === 'user'
  return (
    <div className={isUser ? 'flex justify-end' : ''}>
      <div className={isUser ? 'max-w-[80%] rounded-2xl bg-primary/10 px-4 py-2' : 'max-w-[90%]'}>
        <p className="whitespace-pre-wrap text-sm leading-relaxed">{message.content}</p>
        {message.citations.length > 0 && (
          <div className="mt-3 flex flex-col gap-1.5">
            <p className="text-xs font-medium text-muted-foreground">Sumber</p>
            {message.citations.map((c, i) => (
              <Card key={`${c.document_id ?? c.url}-${i}`}>
                <CardContent className="flex items-center justify-between gap-2 px-3 py-2">
                  <span className="truncate text-xs">
                    [{c.marker}]{' '}
                    {c.url ? (
                      <a href={c.url} target="_blank" rel="noopener noreferrer" className="underline">
                        {c.label}
                      </a>
                    ) : (
                      c.label
                    )}
                    {c.page_no ? ` · hal. ${c.page_no}` : ''}
                  </span>
                  {/* sumber web ditandai domain, biar user tahu ini bukan pustaka terkurasi */}
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {c.url ? new URL(c.url).hostname : `${(c.score * 100).toFixed(0)}%`}
                  </span>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
