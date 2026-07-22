# 08 — Agent Execution Brief

Dokumen ini untuk **agent AI (Claude, session berbeda) yang akan mengeksekusi PLAN ini dari nol.** Baca ini dulu sebelum ngoding. Kalau ada konflik antar dokumen, urutan otoritas: **PRD → Tech Stack → ERD → dokumen lain.**

## 0. Konteks singkat

Bikin **Lexora**, platform AI legal assistant (RAG untuk dokumen hukum Indonesia). Stack: Next 16 + Go + PostgreSQL + Qdrant + n8n + Claude. **Deploy localhost only** untuk sekarang.

**Frontend:** design system SUDAH ADA dari client (folder `frontend/`, Next 16 + Tailwind v4 + shadcn + **bun**). Landing & `/platform` sudah jadi (masih Inggris/US — **lokalisasi ke ID**, jangan rebuild). Halaman app (login, dashboard, chat, KB, workspace) **belum ada → bangun** pakai token design system ([09-DESIGN-SYSTEM](09-DESIGN-SYSTEM.md)). Fungsional dulu; client poles nanti. Run: `bun run dev`.

**Arah masa depan:** 2 update besar — (1) sekarang hukum ID, (2) nanti internasional ("mungkin"). Jangan bangun i18n/jurisdiction sekarang; ikuti seam yang didokumentasikan (Tech Stack §9, ERD catatan). YAGNI sampai fix.

## 1. Aturan main

1. **Ikuti fase berurutan** (05-PLANNING). Jangan lompat. Fase 3 (RAG) butuh Fase 2 (ingestion) jalan.
2. **Satu fase = satu unit kerja yang terverifikasi.** Selesaikan Definition of Done fase itu sebelum lanjut.
3. **Jangan over-engineer.** Tulis kode paling simpel yang jalan. Tidak ada abstraksi spekulatif (interface 1 implementasi, factory, config buat nilai yang gak berubah).
4. **Localhost dulu.** Jangan bikin config produksi, TLS, CI/CD, k8s, Coolify. Itu YAGNI sekarang.
5. **Frontend: jangan rebuild landing.** Pakai `frontend/` yang ada, lokalisasi ke ID. Halaman app baru pakai token & komponen design system ([09](09-DESIGN-SYSTEM.md)) — **jangan hardcode warna**. **bun**, bukan npm/yarn/pnpm.
6. **Kalau ragu / ada keputusan bisnis** (mis. embedding provider), **berhenti dan tanya**, jangan asal pilih yang mengunci arsitektur.

## 2. Konvensi teknis (kunci)

| Hal | Aturan |
|-----|--------|
| ID | `uuid` (v4/v7) semua PK |
| Multi-tenant | Tiap tabel tenant punya `organization_id`. **Setiap query wajib di-scope** via middleware yang inject `org_id` dari JWT. Ini invariant keamanan — jangan dilewat. |
| Migration | **`golang-migrate`**. SQL eksplisit di `backend/db/migrations` (`*.up.sql` + `*.down.sql`). Target: `make migrate-up`, `make migrate-down`. **Bukan** auto-migrate ORM. |
| Run | Backend `make run`, frontend `bun run dev`. Infra (db/qdrant/n8n) `docker compose up -d`. |
| Struktur Go | **Clean Architecture**: `delivery → usecase → domain ← repository`. Dependensi menunjuk ke dalam. Handler tipis, logika di usecase (lihat 04-STRUCTURE §2). |
| Kontrak API | **OpenAPI spec-first**. Edit `openapi/openapi.yaml` dulu → regen server (BE) + client (FE) → implement. Jangan nulis tipe request/response manual. |
| Subdomain | Routing FE per-host via `middleware.ts`: `admin.*`, `app.*`, root=marketing (04-STRUCTURE §4). |
| AI | **Fix Claude.** `pkg/llm` = satu Claude client, model = parameter (Haiku/Sonnet/Opus). Interface `LLM` di domain buat mocking test, bukan multi-provider. Model **diambil dari plan user**, jangan hardcode di usecase. |
| Bahasa jawaban AI | **Ikut bahasa pertanyaan user** (default ID kalau ambigu). Atur lewat instruksi di system prompt (`prompts`), bukan hardcode. |
| Vector | Hidup di Qdrant. Postgres cuma simpan `qdrant_point_id` + metadata chunk. |
| Chat realtime | Go → Qdrant → Claude langsung (streaming SSE). **JANGAN lewat worker ingestion.** |
| Ingestion | **Go background goroutine** (bukan n8n). Upload → simpan file → spawn goroutine → OCR/chunk/embed → Qdrant → update status. |
| SSE format | Stream chat pakai 3 event type: `data: {"token":"..."}` (per token), `data: {"done":true,"citations":[...]}` (akhir), `data: {"error":"..."}` (gagal). FE dan BE wajib konsisten. |
| Token counting | Pakai Claude token counting endpoint (`POST /v1/messages/count_tokens`) **sebelum** kirim ke model — bukan estimasi karakter. Pemakaian = `SUM(token_usage)` user/periode (bukan counter). Detail 06-FLOWS §4. |
| Embedding | Lokal via **Ollama** (`nomic-embed-text`, HTTP `localhost:11434`). `pkg/embedding` = interface + impl Ollama. Ganti provider = ganti impl, dim beda → re-index. |
| Ekstraksi teks | `pdftotext` (exec) untuk PDF ber-teks; **fallback OCR Tesseract** kalau hasil kosong (putusan sering hasil scan). DOCX/TXT langsung. |
| Chunking | Fixed ~800 token, overlap 100, pecah di batas kalimat. Simpan `chunk_index` + `page_no` (citation). Angka default, tandai spot tuning. |
| Ingestion durability | Worker pool terbatas (bukan goroutine tak terbatas). **Recovery saat startup**: scan `uploaded`/`processing` → re-enqueue. Gagal → `status=failed`, jangan diam. |
| Retrieval | Top-k=5, filter Qdrant `organization_id` + `scope`. Skor < threshold → **no-match**: AI jawab jujur tanpa citation palsu, jangan halusinasi (06-FLOWS §2). |
| Privasi chat | Chat privat ke `user_id` pembuatnya. Query scope `organization_id` **dan** `user_id`. Sharing antar-anggota = fase 2. |
| Billing | Subscription **per org** + `seats`. Model dari plan, jangan hardcode. Tambah anggota > seats → tolak. |
| SSE di FE | **Jangan pakai `EventSource`** (tak bisa POST + header). Pakai `fetch` + `ReadableStream`, parse `data:` manual (06-FLOWS §6). |
| Auth di FE | Access token **in-memory** (bukan localStorage). Silent refresh saat app mount + sebelum expiry (06-FLOWS §5). |
| Export | PDF `chromedp` (HTML→PDF), Word `unioffice` (atau HTML+MIME msword lazy). |
| Error | Jangan telan error yang bisa hilangin data (upload gagal, ingestion gagal → set `documents.status=failed`, jangan diam). |
| Secret | Dari `.env`. Jangan hardcode / commit. |
| FE | Folder `frontend/` (bun, Next 16). Pakai token design system, komponen shadcn/base-ui. Jangan hardcode warna/hex. Heading besar `font-serif`. |
| Internasional | Jangan bangun i18n/`jurisdiction` sekarang. Ikuti seam (Tech Stack §9). |
| Security | Ikuti **[10-SECURITY](10-SECURITY.md)**. Non-negotiable: query parameterized + scope `organization_id` (anti-SQLi/IDOR), sanitize output AI (XSS), CORS allowlist + security headers, JWT access-pendek+refresh-rotation, argon2id, rate-limit login/chat. Security **tidak** kena ponytail. |
| Komentar | **Maks 1-3 kata per komentar.** Bahasa manusia, santai, bukan gaya AI kaku. **Tanpa emoji.** Contoh: `// verify jwt`, `// inject org id`. Bukan paragraf. |

## 2.5 Gaya kode & ponytail

- **Ponytail full** aktif. Kode paling simpel yang jalan. Tangga: butuh eksis nggak → stdlib → fitur native → dependency yang udah ada → satu baris → baru kode minimal.
- **Komentar 1-3 kata**, seperlunya, tanpa emoji, bahasa manusia. Jangan jelasin yang udah jelas dari kode.
- Tandai shortcut sengaja dengan `// ponytail: <alasan singkat>`.
- Test cuma untuk logika berisiko (§4), bukan tiap fungsi.

## 3. Definition of Done per fase

Ambil dari 05-PLANNING + 07-INFRA. Ringkas:

- **Fase 0:** infra (`docker compose up`) sehat, backend (`make run`) & frontend (`bun run dev`) jalan, kelima cek §6 di 07-INFRA lolos. Landing render di `localhost:3000`. Super admin ter-seed.
- **Fase 0.5:** Landing + `/platform` tampil bahasa Indonesia, konteks hukum ID (`lang="id"`).
- **Fase 1:** Bisa login lewat UI; org admin bisa tambah user; request user lain org **tidak** bisa baca data org ini (uji isolasi tenant — wajib ada 1 test).
- **Fase 2:** Upload PDF → n8n proses → `documents.status=indexed`, vektor muncul di Qdrant. Uji dengan 1 dokumen nyata.
- **Fase 3:** Tanya pertanyaan → jawaban streaming + minimal 1 citation yang menunjuk chunk sumber benar. `token_usage` tercatat.
- **Fase 4:** Limit token soft(80%)/hard(100%) bekerja; dashboard tampil; export PDF/Word jalan.
- **Fase 5:** Audit log terisi; siap (belum wajib) dipindah ke server.

## 4. Test — secukupnya (jangan berlebihan)

Wajib ada test kecil untuk **logika berisiko**, tidak untuk semua fungsi:
- Isolasi tenant (query lintas-org harus kosong/403).
- Perhitungan limit token (boundary 80% / 100%).
- Parsing citation dari hasil RAG.

Tidak perlu framework berat, fixture, atau coverage 100%. Satu test yang gagal kalau logikanya rusak — cukup.

## 5. Alur kerja yang disarankan

1. Baca 00→08 sekali.
2. Kerjakan Fase 0, verifikasi, commit.
3. Lanjut fase berikut satu per satu, verifikasi tiap Definition of Done.
4. Update checkbox di 05-PLANNING saat fase kelar.
5. Kalau nemu keputusan yang belum dikunci (ditandai "perlu konfirmasi client" di dokumen) → tanya, jangan tebak.

## 6. Keputusan (mostly fix)

Semua keputusan besar sudah dikunci (lihat `question.md` rekap). Sisa detail kecil, pakai default, jangan berhenti:

| Keputusan | Default |
|-----------|---------|
| Embedding | lokal (`nomic-embed-text`, dim 768). Fallback Voyage kalau berat → re-index |
| Model Demo tier | Claude Haiku (murah); bisa ganti GPT gratis kalau client mau |
| Payment gateway | manual invoice dulu, gateway (Midtrans/Xendit) ditunda |
| Retensi/purge chat | soft delete, purge belakangan |
| Bilingual landing | ID dulu; toggle EN = bonus kalau sempat |

Jangan biarkan detail kecil memblok kerja yang tak bergantung padanya.
