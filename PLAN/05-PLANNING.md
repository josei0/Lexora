# 05 — Planning / Roadmap

Bertahap. Tiap fase menghasilkan sesuatu yang jalan. **Semua di localhost.** Infra (postgres/qdrant/n8n) via docker-compose; backend `make run`, frontend `bun run dev`. **Frontend dari client** (folder `frontend/`) — landing tinggal dilokalisasi, halaman app dibangun pakai design system ([09](09-DESIGN-SYSTEM.md)), fungsional dulu. Detail eksekusi & guardrail: [08-AGENT-BRIEF](08-AGENT-BRIEF.md).

## Fase 0 — Fondasi

- [ ] `docker-compose.yml`: postgres, qdrant, n8n (infra saja)
- [ ] Skeleton backend Go (health check, config, structured logging), jalan `make run`
- [ ] `openapi/openapi.yaml` awal + `make gen`
- [ ] Frontend client jalan native (`bun install` → `bun run dev`), landing render
- [ ] Migration awal: `organizations`, `users`, `memberships`
- [ ] Seed super admin

**Selesai =** infra up, backend `:8080` & frontend `:3000` jalan, landing render.

## Fase 0.5 — Lokalisasi landing (ringan, paralel)

- [ ] `frontend/app/layout.tsx`: `lang="en"` → `id`, metadata ID
- [ ] Copy landing + `/platform` → Indonesia + konteks hukum ID (aku draft dulu)
- [ ] 4 modul disesuaikan ke fitur nyata (riset/analisis/draft/KB)

**Selesai =** landing tampil dalam bahasa Indonesia, konteks hukum ID.

## Fase 1 — Auth & Multi-tenant

- [ ] Login email+password, JWT (access 15m + refresh rotation via tabel `refresh_tokens`), argon2id
- [ ] Middleware inject `organization_id` dari JWT (anti-IDOR)
- [ ] Security dasar: CORS allowlist, security headers, rate-limit login, error login generik, body-size limit
- [ ] CRUD organization (super admin) + tambah anggota + password sementara `must_change_password` (org admin)
- [ ] Role guard: super_admin / org_admin / member
- [ ] **FE:** login + app shell (sidebar) + silent refresh saat app mount (access in-memory)

**Selesai =** login lewat UI, refresh halaman tetap login, data ter-scope per org, uji lintas-org gagal (test IDOR). Lihat [10-SECURITY](10-SECURITY.md).

## Fase 2 — Knowledge Base & Ingestion (Go worker)

- [ ] Upload dokumen (PDF/DOCX/TXT, ≤20MB) → simpan ke disk + status `uploaded`
- [ ] Go background goroutine: OCR → chunking → embedding → upsert Qdrant → update status `indexed`
- [ ] Error handling: gagal → set `documents.status=failed`, jangan diam
- [ ] Tabel `documents`, `document_chunks`
- [ ] **FE:** halaman Knowledge Base (upload, list, status indexing)

> Tidak pakai n8n di MVP — ingestion cukup goroutine Go. n8n masuk update besar #1 untuk ingestion sumber eksternal terjadwal (JDIHN/BPK).
> ⚠️ Kunci **provider embedding** (lokal vs Voyage) sebelum mulai — nentuin `EMBEDDING_DIM` di Qdrant. Lihat `question.md` §1.

**Selesai =** upload dokumen, otomatis terindeks di Qdrant.

## Fase 3 — Chat + RAG + Citation

- [ ] Chat CRUD + workspace default
- [ ] RAG: search Qdrant → susun konteks → Claude (streaming SSE)
- [ ] Citation dari chunk yang dipakai (`citations`)
- [ ] `token_usage` per pesan
- [ ] Chat: rename, delete (soft), search, pin
- [ ] **FE:** halaman Chat (streaming, tampil citation) + Workspace

**Selesai =** user tanya, dapat jawaban + sumber, token tercatat.

## Fase 4 — Subscription & Dashboard

- [ ] `plans`: Demo (gratis, Haiku) & Pro ($17/seat, Sonnet). `subscriptions` **per org** + `seats`
- [ ] Model dipilih dari plan (jangan hardcode di usecase). Tambah anggota > seats ditolak
- [ ] Limit token: pemakaian = `SUM(token_usage)` per user/periode. Soft 80% (warning) → hard block 100%
- [ ] Dashboard: chat hari ini (WIB), token, dokumen, workspace, aktivitas
- [ ] Prompt registry (super admin edit)
- [ ] Export PDF (chromedp) / Word (unioffice)
- [ ] **FE:** halaman Dashboard (statistik pakai `chart-*` token), panel admin (super/org)

**Selesai =** MVP lengkap, siap demo ke client.

## Fase 5 — Hardening (localhost)

- [ ] Audit log lengkap
- [ ] Monitoring penggunaan (super admin)
- [ ] Review keamanan pakai checklist [10-SECURITY](10-SECURITY.md); `govulncheck` + `bun audit`
- [ ] Siapkan agar mudah dipindah ke server (env terpisah) — **belum deploy Coolify**

**Selesai =** MVP lengkap jalan di localhost, siap demo ke client.

> Deploy Coolify/cloud = pekerjaan setelah MVP disetujui client. Bukan bagian roadmap ini.

---

## Setelah MVP

**Update besar #1 (lanjutan fitur ID):**
- Sumber eksternal (JDIHN/BPK/Peraturan.go.id/Hukumonline) via n8n scheduled
- Flow hukum terdedikasi: analisis perkara, kronologi, persiapan sidang
- Template draft dokumen (gugatan, kontrak)
- Editor dokumen in-app
- Tier Premium (Claude Opus)
- Payment gateway (Midtrans/Xendit)

**Update besar #2 (internasional — "mungkin"):**
- Tambah kolom `jurisdiction` di `documents` + filter Qdrant
- i18n UI (framework, kalau memang jadi)
- Model/embedding multibahasa
- Belum dijadwalkan; cuma seam-nya yang didokumentasikan sekarang (Tech Stack §9).

---

## Urutan kritis (dependency)

```
Fase 0 → 0.5 (landing, paralel) 
   └→ 1 → 2 → 3 → 4 → 5
              ↑
     (3 butuh 2: RAG butuh dokumen terindeks)
```

- Jangan lompat ke Chat/RAG sebelum ingestion jalan — nggak ada yang bisa di-retrieve.
- Fase 0.5 (lokalisasi landing) bisa jalan kapan aja setelah Fase 0, nggak blokir backend.
