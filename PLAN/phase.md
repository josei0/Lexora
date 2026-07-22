# Phase — Lexora Execution Order

Urutan kerja konkret. Tiap fase = unit terverifikasi sebelum lanjut.
Dependency kritis: Fase 3 butuh Fase 2 selesai (RAG butuh dokumen terindeks).

---

## Fase 0 — Fondasi ✅ SELESAI (2026-07-22)
**Target:** infra jalan, backend & frontend bisa diakses, landing render.

| # | Task | File/Lokasi |
|---|------|-------------|
| 0.1 | ✅ `docker-compose.yml` (postgres, qdrant) | root |
| 0.2 | ✅ `.env.example` + `.env` (isi `ANTHROPIC_API_KEY`) | root |
| 0.3 | ✅ Skeleton Go: `cmd/server/main.go`, `config/config.go`, health endpoint `/healthz` | backend |
| 0.4 | ✅ `backend/Makefile` (`make run`, `make migrate-up`, `make migrate-down`, `make gen`, `make test`) | backend |
| 0.5 | ✅ Migration awal (`golang-migrate`): `organizations`, `users`, `memberships` | `backend/db/migrations/` |
| 0.6 | ✅ Seed super admin | `backend/db/seed/` |
| 0.7 | ✅ `openapi/openapi.yaml` skeleton + `make gen` | openapi |
| 0.8 | ✅ `bun install` di `frontend/`, pastikan `bun run dev` jalan | frontend |

**DoD:** 4 cek di 07-INFRA §5 semua hijau. Landing render di `localhost:3000`.

**Status verifikasi (2026-07-22):**
- ✅ Backend `/healthz` → `{"status":"ok"}` (via `make run`)
- ✅ Frontend landing `localhost:3000` → HTTP 200 (`bun run dev`)
- ✅ Postgres connect + migrate up/down + seed super admin (`admin@lexora.id`)
- ✅ `make gen`/`test`/`migrate-up`/`migrate-down`/`seed` semua jalan
- ⚠️ Qdrant `:6333/healthz` **belum diverifikasi** — Docker belum terpasang di host. Tidak blokir Fase 1 (Qdrant baru dipakai Fase 2). Jalankan `docker compose up -d` sebelum Fase 2.

---

## Fase 0.5 — Lokalisasi Landing ✅ SELESAI (2026-07-22)
**Target:** landing tampil bahasa Indonesia, konteks hukum ID.

| # | Task | File/Lokasi |
|---|------|-------------|
| 0.5.1 | ✅ `lang="en"` → `lang="id"`, metadata title/desc ke ID | `frontend/app/layout.tsx` |
| 0.5.2 | ✅ Copy hero, tagline, fitur → Indonesia + konteks hukum ID | komponen landing |
| 0.5.3 | ✅ 4 modul di `platform-modules.tsx` → Riset hukum, Analisis kontrak, Studio dokumen, Pustaka privat | `frontend/components/platform/` |
| 0.5.4 | ✅ Ganti referensi hukum US (Shepardizing, jurisdiction, SOC 2) → ID (UU, PP, Putusan, PKPU) | seluruh landing |

**DoD:** landing + `/platform` tampil ID, tidak ada teks US/EN yang tersisa.

**Status verifikasi (2026-07-22):**
- ✅ `layout.tsx`: `lang="id"` + metadata ID
- ✅ Landing (`hero`, `capabilities`, `stats`, `workflow`, `practice-areas`, `cta`, `site-header`, `site-footer`) full ID
- ✅ `/platform` (`platform-hero`, `platform-modules`, `platform-security`) full ID
- ✅ Bidang praktik → konteks ID (Kepailitan & PKPU, PDP, Pasar Modal, dsb.)
- ✅ Grep sisa teks EN → nol (hanya identifier kode/anchor yang cocok)
- ℹ️ Folder komponen aktual `components/` & `components/platform/` (plan menyebut `marketing/`; tidak ada, disesuaikan)

---

## Fase 1 — Auth & Multi-tenant
**Target:** login lewat UI, data ter-scope per org, isolasi tenant terverifikasi.

| # | Task | File/Lokasi |
|---|------|-------------|
| 1.1 | Domain: `user`, `organization`, `membership` + interface repo | `internal/domain/` |
| 1.2 | Migration: `workspaces`, `chats`, `messages`, `plans`, `subscriptions` (org-level + seats), `audit_logs` + index `organization_id` semua tabel tenant | `backend/db/migrations/` |
| 1.3 | `auth_usecase`: login (argon2id), JWT issue (access 15m + refresh rotation), `must_change_password` flow | `internal/usecase/` |
| 1.4 | `pkg/jwt`: sign, verify, klaim `user_id/org_id/role` | `backend/pkg/jwt/` |
| 1.5 | Middleware: `auth` (verify JWT), `tenant` (inject org_id), `rbac` (guard role per route — matriks 10-SECURITY §RBAC) | `internal/delivery/http/middleware/` |
| 1.6 | Migration: `refresh_tokens` (hash, expires_at, revoked_at). Rotation: rotate tiap refresh, revoke saat logout | `backend/db/migrations/` |
| 1.7 | Middleware: `cors` (allowlist), `ratelimit` (login 5/mnt/IP + 10/mnt/akun), security headers, body-size limit | middleware |
| 1.8 | Handler: `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`. Error login generik | `handler/auth_handler.go` |
| 1.9 | Handler: CRUD organization (super admin), tambah anggota + password sementara `must_change_password`, set role anggota (org admin) | `handler/organization_handler.go` |
| 1.10 | FE: login + app shell (sidebar) + **silent refresh saat app mount** (access token in-memory, lihat 06-FLOWS §5) | `frontend/app/(auth)/login/`, `frontend/app/app/layout.tsx` |
| 1.11 | Test: isolasi tenant (query lintas-org → 403/kosong) + RBAC (member akses route admin → 403) | `backend/internal/usecase/*_test.go` |

**DoD:** login lewat UI berhasil; refresh halaman tetap login (silent refresh); org admin bisa tambah user; request lintas-org gagal (test lolos).

---

## Fase 2 — Knowledge Base & Ingestion
**Target:** upload dokumen → otomatis terindeks di Qdrant.

> Konfirmasi embedding provider sebelum mulai (default: lokal `nomic-embed-text` dim 768).
> Client sediakan 2-3 PDF hukum contoh untuk uji end-to-end.

| # | Task | File/Lokasi |
|---|------|-------------|
| 2.1 | Domain: `document`, `document_chunk` + `VectorRepository` + `Embedder` interface | `internal/domain/document.go` |
| 2.2 | `pkg/embedding`: Ollama HTTP client (`nomic-embed-text`) | `backend/pkg/embedding/` |
| 2.3 | Ekstraksi teks: `pdftotext` via exec; fallback OCR Tesseract kalau hasil kosong. DOCX + TXT | `backend/pkg/extract/` |
| 2.4 | `document_usecase`: validasi MIME (sniff isi) + ≤20MB → simpan disk → insert (status=uploaded) → kirim ke worker pool | `internal/usecase/document_usecase.go` |
| 2.5 | `pkg/storage`: local file impl (`STORAGE_DIR/<org_id>/<doc_id>/`) | `backend/pkg/storage/` |
| 2.6 | Handler: `POST /documents` (upload → 202), `GET /documents` (list + status, paginated) | `handler/document_handler.go` |
| 2.7 | Worker pool (N=3) + **recovery saat startup** (scan `uploaded`/`processing` → re-enqueue): ensure collection → ekstrak → chunk (800/overlap100) → embed → upsert Qdrant → status. Gagal → status=failed | `internal/usecase/ingestion_usecase.go` |
| 2.8 | `repository/qdrant/vector_repo.go`: ensure collection (idempotent), upsert, search | `internal/repository/qdrant/` |
| 2.9 | FE: halaman Knowledge Base (upload, list, status badge, retry kalau failed) | `frontend/app/app/knowledge-base/` |

**DoD:** upload 1 PDF nyata → `documents.status=indexed`, vektor muncul di Qdrant dashboard. Restart backend saat processing → dokumen tetap kelar (recovery jalan).

---

## Fase 3 — Chat + RAG + Citation
**Target:** tanya pertanyaan → jawaban streaming + citation sumber.

| # | Task | File/Lokasi |
|---|------|-------------|
| 3.1 | Domain: `chat`, `message`, `citation`, `token_usage` | `internal/domain/` |
| 3.2 | `pkg/llm`: satu Claude client, model = parameter | `backend/pkg/llm/` |
| 3.3 | `rag_usecase`: embed query → search top-k (filter org_id + scope) → filter skor ≥ threshold → susun konteks → Claude stream → simpan. **No-match → jawab jujur tanpa citation palsu** | `internal/usecase/rag_usecase.go` |
| 3.4 | `billing_usecase`: `count_tokens` (pre-flight), pemakaian = `SUM(token_usage)` user periode berjalan vs `monthly_token_limit` (soft 80% / hard 100%), insert `token_usage` dari `usage` response | `internal/usecase/billing_usecase.go` |
| 3.5 | Handler: `POST /chats`, `POST /chats/{id}/messages` (SSE — format `token`/`done`/`error`, 06-FLOWS §2), rename, delete (soft), search, pin. Chat scope `org_id` + `user_id` (privat) | `handler/chat_handler.go` |
| 3.6 | System prompt: bahasa jawaban ikut pertanyaan (default ID); larang jawab tanpa sumber KB | `prompts` table seed |
| 3.7 | FE: halaman Chat — **konsumsi SSE via `fetch` + ReadableStream** (bukan EventSource, 06-FLOWS §6), citation card, token meter | `frontend/app/app/chat/` |
| 3.8 | FE: halaman Workspace | `frontend/app/app/workspaces/` |
| 3.9 | Test: parsing citation dari RAG; limit token boundary 80%/100%; no-match tidak halusinasi | `backend/internal/usecase/*_test.go` |

**DoD:** tanya pertanyaan → jawaban streaming + minimal 1 citation menunjuk chunk benar; `token_usage` tercatat.

---

## Fase 4 — Subscription & Dashboard
**Target:** MVP lengkap, siap demo ke client.

| # | Task | File/Lokasi |
|---|------|-------------|
| 4.1 | Seed `plans`: Demo (Haiku, gratis) + Pro (Sonnet, $17/seat). `monthly_token_limit` per seat | `backend/db/seeds/` |
| 4.2 | `subscription_usecase`: subscription **per org** + `seats`. Model dari plan (jangan hardcode). Tambah anggota tolak kalau > seats | `internal/usecase/subscription_usecase.go` |
| 4.3 | `dashboard_usecase`: chat hari ini (timezone WIB), token used, jumlah dokumen, workspace, aktivitas | `internal/usecase/dashboard_usecase.go` |
| 4.4 | `prompt_usecase`: CRUD prompt (super admin edit tanpa deploy) | `internal/usecase/prompt_usecase.go` |
| 4.5 | Export chat: PDF via `chromedp` (render HTML→PDF), Word via `unioffice` (atau HTML+MIME msword lazy) | `internal/usecase/export_usecase.go` + handler |
| 4.6 | FE: halaman Dashboard (chart-* token, statistik) | `frontend/app/app/dashboard/` |
| 4.7 | FE: panel admin — users, organizations, ai-models, prompts, billing | `frontend/app/admin/` |
| 4.8 | FE: halaman Settings (user profile, subscription info, ganti password) | `frontend/app/app/settings/` |

**DoD:** limit token soft/hard bekerja; dashboard tampil data nyata; export PDF/Word jalan; tambah anggota > seats ditolak.

---

## Fase 5 — Hardening
**Target:** MVP siap demo, audit log terisi, checklist security lolos.

| # | Task | File/Lokasi |
|---|------|-------------|
| 5.1 | Audit log lengkap (semua aksi penting tercatat di `audit_logs`) | `internal/usecase/audit_usecase.go` |
| 5.2 | Monitoring penggunaan di panel super admin | `frontend/app/admin/` |
| 5.3 | Review checklist 10-SECURITY: semua 18 ancaman diverifikasi (termasuk matriks RBAC + test lintas-role/user) | — |
| 5.4 | `govulncheck ./...` (Go) + `bun audit` (FE) — nol critical | dev workflow |
| 5.5 | Pastikan `.env` tidak ter-commit, secret aman | git check |

**DoD:** audit log terisi; `govulncheck` + `bun audit` bersih; checklist 10-SECURITY semua centang.

---

## Dependency tree

```
Fase 0
  └── Fase 0.5 (paralel, kapan saja setelah 0)
  └── Fase 1
        └── Fase 2
              └── Fase 3
                    └── Fase 4
                          └── Fase 5
```

Jangan lompat fase. Fase 3 tidak bisa jalan tanpa dokumen terindeks dari Fase 2.

---

## Prasyarat sebelum Fase 2 (checklist, bukan keputusan terbuka)

Semua keputusan sudah dikunci (lihat [12-QUESTIONS](12-QUESTIONS.md)). Yang perlu disiapkan sebelum mulai Fase 2:

| Hal | Aksi |
|-----|------|
| Ollama + `nomic-embed-text` | `ollama pull nomic-embed-text` di host (native) |
| poppler-utils + tesseract-ocr | Terinstall di host untuk ekstraksi teks + OCR |
| PDF hukum contoh | Client kirim 2-3 file (UU/putusan) untuk uji end-to-end |

> Domain `lexora.id` cuma perlu dikonfirmasi sebelum config prod (CORS/email) — tidak blokir MVP localhost.
