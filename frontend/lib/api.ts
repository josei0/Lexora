// api client: access token in-memory (anti-XSS), refresh via HttpOnly cookie

const BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

let accessToken: string | null = null

export function setAccessToken(t: string | null) {
  accessToken = t
}
export function getAccessToken() {
  return accessToken
}

export type Tokens = {
  access_token: string
  token_type: string
  expires_in: number
  must_change_password?: boolean
}

export class ApiError extends Error {
  code: string
  status: number
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

async function parseError(res: Response): Promise<ApiError> {
  let code = 'error'
  let message = 'terjadi kesalahan'
  try {
    const body = await res.json()
    if (body?.error) {
      code = body.error.code ?? code
      message = body.error.message ?? message
    }
  } catch {
    // non-json body
  }
  return new ApiError(res.status, code, message)
}

// login: sets refresh cookie, returns tokens
export async function login(email: string, password: string): Promise<Tokens> {
  const res = await fetch(`${BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) throw await parseError(res)
  const tok: Tokens = await res.json()
  accessToken = tok.access_token
  return tok
}

// refresh: rotate cookie + new access token. throws on 401
// dedup concurrent callers so token rotation isn't raced (strict-mode mount, parallel 401s)
let inflight: Promise<Tokens> | null = null

export async function refresh(): Promise<Tokens> {
  if (inflight) return inflight
  inflight = (async () => {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!res.ok) throw await parseError(res)
    const tok: Tokens = await res.json()
    accessToken = tok.access_token
    return tok
  })()
  try {
    return await inflight
  } finally {
    inflight = null
  }
}

export async function logout(): Promise<void> {
  try {
    await fetch(`${BASE}/auth/logout`, { method: 'POST', credentials: 'include' })
  } finally {
    accessToken = null
  }
}

// authed fetch: attach bearer, retry once after silent refresh on 401
export async function api<T = unknown>(path: string, init: RequestInit = {}): Promise<T> {
  const isForm = init.body instanceof FormData
  const doFetch = () =>
    fetch(`${BASE}${path}`, {
      ...init,
      credentials: 'include',
      headers: {
        ...(init.body && !isForm ? { 'Content-Type': 'application/json' } : {}),
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
        ...init.headers,
      },
    })

  let res = await doFetch()
  if (res.status === 401) {
    // reactive silent refresh, then retry once
    await refresh()
    res = await doFetch()
  }
  if (res.status === 204) return undefined as T
  if (!res.ok) throw await parseError(res)
  return res.json() as Promise<T>
}

export type Document = {
  id: string
  file_name: string
  mime_type: string
  file_size: number
  scope?: string
  status: 'uploaded' | 'processing' | 'indexed' | 'failed'
  error?: string
  created_at: string
}

export function listDocuments() {
  return api<Document[]>('/documents')
}

// multipart upload; browser sets the boundary itself
export function uploadDocument(file: File) {
  const form = new FormData()
  form.append('file', file)
  return api<Document>('/documents', { method: 'POST', body: form })
}

export type Chat = {
  id: string
  title: string
  is_pinned: boolean
  created_at: string
  updated_at: string
}

export type Citation = {
  marker: number
  label: string
  document_id?: string
  page_no?: number
  score: number
}

export type Message = {
  id: string
  role: 'user' | 'assistant'
  content: string
  model?: string
  created_at: string
  citations: Citation[]
}

export function listChats(search = '') {
  const q = search ? `?search=${encodeURIComponent(search)}` : ''
  return api<Chat[]>(`/chats${q}`)
}

export function createChat(title = '') {
  return api<Chat>('/chats', { method: 'POST', body: JSON.stringify({ title }) })
}

export function chatMessages(chatId: string) {
  return api<Message[]>(`/chats/${chatId}/messages`)
}

export function renameChat(chatId: string, title: string) {
  return api<void>(`/chats/${chatId}`, { method: 'PATCH', body: JSON.stringify({ title }) })
}

export function pinChat(chatId: string, isPinned: boolean) {
  return api<void>(`/chats/${chatId}`, { method: 'PATCH', body: JSON.stringify({ is_pinned: isPinned }) })
}

export function deleteChat(chatId: string) {
  return api<void>(`/chats/${chatId}`, { method: 'DELETE' })
}

type StreamHandlers = {
  onToken: (token: string) => void
  onDone: (citations: Citation[], messageId: string) => void
  onError: (message: string) => void
}

// SSE lewat fetch + ReadableStream (EventSource tidak bisa kirim bearer)
export async function askStream(chatId: string, content: string, h: StreamHandlers) {
  const doFetch = () =>
    fetch(`${BASE}/chats/${chatId}/messages`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      },
      body: JSON.stringify({ content }),
    })

  let res = await doFetch()
  if (res.status === 401) {
    await refresh()
    res = await doFetch()
  }
  if (!res.ok || !res.body) throw await parseError(res)

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    const frames = buffer.split('\n\n')
    buffer = frames.pop() ?? ''
    for (const frame of frames) {
      const line = frame.trim()
      if (!line.startsWith('data:')) continue
      try {
        const evt = JSON.parse(line.slice(5).trim())
        if (evt.error) h.onError(evt.error)
        else if (evt.done) h.onDone(evt.citations ?? [], evt.message_id)
        else if (evt.token) h.onToken(evt.token)
      } catch {
        // frame belum utuh
      }
    }
  }
}

// ── Phase 4 types ──────────────────────────────────────────────────────────

export type Plan = {
  id: string
  code: string
  name: string
  model: string
  price_idr: number
  monthly_token_limit: number
  is_active: boolean
}

export type SubscriptionView = {
  id: string
  organization_id: string
  plan_id: string
  seats: number
  created_at: string
  updated_at: string
  plan: Plan
}

export type DashboardStats = {
  chats_today: number
  tokens_month: number
  token_limit: number
  docs_indexed: number
  docs_total: number
  members: number
  seats: number
}

export type Quota = {
  limit: number
  used: number
  remaining: number
  soft: boolean
  hard: boolean
}

export type PromptData = {
  key: string
  content: string
  updated_at: string
}

// ── Phase 4 API calls ──────────────────────────────────────────────────────

export function getDashboard() {
  return api<DashboardStats>('/dashboard')
}

export function getQuota() {
  return api<Quota>('/quota')
}

export function listPlans() {
  return api<Plan[]>('/plans')
}

export function getSubscription() {
  return api<SubscriptionView | null>('/subscription')
}

export function assignSubscription(orgId: string, planCode: string, seats: number) {
  return api<SubscriptionView>('/subscription', {
    method: 'PUT',
    body: JSON.stringify({ org_id: orgId, plan_code: planCode, seats }),
  })
}

export function getPrompt(key: string) {
  return api<PromptData>(`/prompts/${key}`)
}

export function setPrompt(key: string, content: string) {
  return api<void>(`/prompts/${key}`, { method: 'PUT', body: JSON.stringify({ content }) })
}

export function exportChatUrl(chatId: string, format: 'word' | 'pdf') {
  return `${BASE}/chats/${chatId}/export?format=${format}`
}

export async function exportChat(chatId: string, format: 'word' | 'pdf') {
  const res = await fetch(exportChatUrl(chatId, format), {
    credentials: 'include',
    headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {},
  })
  if (!res.ok) throw await parseError(res)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = res.headers.get('Content-Disposition')?.match(/filename="(.+)"/)?.[1] ?? `chat.${format === 'word' ? 'doc' : 'html'}`
  a.click()
  URL.revokeObjectURL(url)
}
