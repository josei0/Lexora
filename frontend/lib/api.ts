// api client: access token in-memory (anti-XSS), refresh via HttpOnly cookie

const BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

let accessToken: string | null = null

export function setAccessToken(t: string | null) {
  accessToken = t
}
export function getAccessToken() {
  return accessToken
}

export type Role = { system: string; org: string }

function jwtClaims(): Record<string, unknown> | null {
  if (!accessToken) return null
  try {
    const b64 = accessToken.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
    const pad = b64.length % 4 ? '='.repeat(4 - (b64.length % 4)) : ''
    return JSON.parse(atob(b64 + pad))
  } catch {
    return null
  }
}

// decode klaim role dari JWT access token (untuk gating UI, bukan otorisasi)
export function currentRole(): Role | null {
  const c = jwtClaims()
  if (!c) return null
  return { system: (c.sys_role as string) ?? 'none', org: (c.org_role as string) ?? '' }
}

export function currentUserId(): string {
  return (jwtClaims()?.sub as string) ?? ''
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
  }
  return new ApiError(res.status, code, message)
}

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

// login admin 2 langkah: password -> (enroll | mfa) -> token
export type AdminLoginStep = {
  enroll_required?: boolean
  mfa_required?: boolean
  otpauth_url?: string
}

export type AdminTokens = Tokens & { recovery_codes?: string[] }

async function postAdminAuth<T>(path: string, body: object): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(body),
  })
  if (!res.ok) throw await parseError(res)
  return res.json() as Promise<T>
}

export async function adminLogin(email: string, password: string): Promise<AdminLoginStep> {
  return postAdminAuth<AdminLoginStep>('/auth/admin/login', { email, password })
}

export async function adminEnroll(email: string, password: string, code: string): Promise<AdminTokens> {
  const tok = await postAdminAuth<AdminTokens>('/auth/admin/enroll', { email, password, code })
  accessToken = tok.access_token
  return tok
}

export async function adminVerify(email: string, password: string, code: string): Promise<Tokens> {
  const tok = await postAdminAuth<Tokens>('/auth/admin/verify', { email, password, code })
  accessToken = tok.access_token
  return tok
}

// anggota org (org_admin only)
export type Member = {
  user_id: string
  email: string
  full_name: string
  role: 'org_admin' | 'member'
  is_active: boolean
}

export type NewMember = {
  user_id: string
  email: string
  temp_password: string
}

// sumber web -> pustaka (org_admin)
export type WebResult = { title: string; url: string; snippet: string }
export type WebPreview = { title: string; text: string; chars: number }
export type WebCandidate = { url: string; hits: number; last_at: string }

export function searchWebSources(query: string, limit = 4): Promise<WebResult[]> {
  return api<WebResult[]>('/web-sources/search', {
    method: 'POST',
    body: JSON.stringify({ query, limit }),
  })
}

export function previewWebSource(url: string): Promise<WebPreview> {
  return api<WebPreview>('/web-sources/preview', { method: 'POST', body: JSON.stringify({ url }) })
}

export function ingestWebSource(url: string): Promise<{ id: string; file_name: string }> {
  return api<{ id: string; file_name: string }>('/web-sources/ingest', {
    method: 'POST',
    body: JSON.stringify({ url }),
  })
}

export function listWebCandidates(): Promise<WebCandidate[]> {
  return api<WebCandidate[]>('/web-sources/candidates')
}

// tagihan (org_admin): riwayat invoice org sendiri
export type Invoice = {
  id: string
  type: 'subscription' | 'topup'
  package_code?: string
  seats: number
  amount_idr: number
  period_start: string
  period_end: string
  status: 'pending' | 'paid' | 'expired' | 'void'
  paid_at?: string
  created_at: string
}

export function listInvoices(): Promise<Invoice[]> {
  return api<Invoice[]>('/invoices')
}

// paket top-up: harga final ditentukan server, FE hanya kirim code
export type TopupPackage = { code: 'small' | 'large'; label: string; tokens: number; price_idr: number }

export const TOPUP_PACKAGES: TopupPackage[] = [
  { code: 'small', label: '500 ribu token', tokens: 500_000, price_idr: 79_000 },
  { code: 'large', label: '1 juta token', tokens: 1_000_000, price_idr: 149_000 },
]

// buat invoice top-up pending; pelunasan manual super_admin (fase 12: gateway)
export function createTopup(packageCode: 'small' | 'large'): Promise<Invoice> {
  return api<Invoice>('/billing/topup', {
    method: 'POST',
    body: JSON.stringify({ package_code: packageCode }),
  })
}

export function listMembers(): Promise<Member[]> {
  return api<Member[]>('/members')
}

export function addMember(email: string, fullName: string, role: Member['role']): Promise<NewMember> {
  return api<NewMember>('/members', {
    method: 'POST',
    body: JSON.stringify({ email, full_name: fullName, role }),
  })
}

export function updateMember(
  userId: string,
  patch: { role?: Member['role']; is_active?: boolean },
): Promise<Member> {
  return api<Member>(`/members/${userId}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

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
  url?: string // terisi = sumber web
}

export type Message = {
  id: string
  role: 'user' | 'assistant'
  content: string
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

// soft = pemakaian >= 80%, degraded = jatah High habis, jawaban lewat Normal
export type QuotaFlags = { soft?: boolean; degraded?: boolean }
// skipped: 'plan' | 'quota' | 'failed'
export type WebFlags = { used?: boolean; skipped?: string }

type StreamHandlers = {
  onToken: (token: string) => void
  onStatus?: (status: string) => void
  onDone: (citations: Citation[], messageId: string, quota?: QuotaFlags, web?: WebFlags) => void
  onError: (message: string) => void
}

// SSE lewat fetch + ReadableStream (EventSource tidak bisa kirim bearer)
export async function askStream(
  chatId: string,
  content: string,
  files: File[],
  h: StreamHandlers,
  webSearch = false,
) {
  const doFetch = () => {
    // multipart: browser set boundary sendiri, jangan set Content-Type
    const form = new FormData()
    form.append('content', content)
    // lampiran menang: backend abaikan web_search kalau ada file
    if (webSearch && files.length === 0) form.append('web_search', 'true')
    for (const f of files) form.append('files', f)
    return fetch(`${BASE}/chats/${chatId}/messages`, {
      method: 'POST',
      credentials: 'include',
      headers: { ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}) },
      body: form,
    })
  }

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
        else if (evt.done) h.onDone(evt.citations ?? [], evt.message_id, evt.quota, evt.web)
        else if (evt.status) h.onStatus?.(evt.status)
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
  price_idr: number
  monthly_token_limit: number
  is_active: boolean
}

// label tier ke user; nama model sengaja tidak pernah dikirim backend
export type Tier = 'high' | 'normal'

export function tierLabel(t: Tier | undefined) {
  return t === 'normal' ? 'AI Normal' : 'AI High'
}

export type SubscriptionView = {
  id: string
  organization_id: string
  plan_id: string
  seats: number
  current_period_end?: string
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

export type SubStatus = 'active' | 'past_due' | 'expired'

export type Quota = {
  limit: number
  used: number
  remaining: number
  soft: boolean
  hard: boolean
  overflow: boolean
  tier: Tier
  status: SubStatus
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

export type AuditLog = {
  id: string
  org_id?: string
  user_id?: string
  action: string
  resource_id?: string
  ip: string
  created_at: string
}

export function listAuditLogs(limit = 100) {
  return api<AuditLog[]>(`/audit-logs?limit=${limit}`)
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
