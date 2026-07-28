# MindLaw

Platform asisten hukum berbasis AI untuk praktisi hukum Indonesia. Upload dokumen hukum (UU, PP, putusan), lalu tanya langsung — jawaban disusun dari isi dokumen dengan kutipan sumber.

## Arsitektur

```
frontend/   Next.js 16 (App Router, TypeScript, Tailwind)
backend/    Go 1.26 (clean architecture: domain → repo → usecase → handler)
openapi/    OpenAPI 3.0 spec + codegen
```

Infra: PostgreSQL 18 (native host), Qdrant (Docker), Maia Router (embedding + chat via OpenAI-compatible API).

---

## Prasyarat

- Go 1.26+
- Node.js 20+ dan [Bun](https://bun.sh)
- PostgreSQL 18 berjalan di `localhost:5432`
- Docker (untuk Qdrant)
- `pdftotext` dari poppler-utils (`scoop install poppler` di Windows)
- `golang-migrate` CLI (`go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`)
- API key [Maia Router](https://maiarouter.ai) (embedding + chat)

---

## Setup

### 1. Konfigurasi environment

```bash
cp .env.example .env
```

Isi nilai berikut di `.env`:

| Variabel | Keterangan |
|---|---|
| `DATABASE_URL` | Koneksi PostgreSQL |
| `JWT_SECRET` | String acak, minimal 32 karakter |
| `MAIA_API_KEY` | API key Maia Router |
| `SUPERADMIN_PASSWORD` | Password awal super admin |

### 2. Jalankan Qdrant

```bash
docker compose up -d
```

### 3. Buat database PostgreSQL

```sql
CREATE USER applaw WITH PASSWORD 'change_me_locally';
CREATE DATABASE applaw OWNER applaw;
```

Sesuaikan kredensial dengan `DATABASE_URL` di `.env`.

### 4. Migrasi dan seed

```bash
cd backend
make migrate-up
make seed
```

`make seed` membuat akun super admin (`SUPERADMIN_EMAIL` / `SUPERADMIN_PASSWORD`) dan mengisi tabel `plans` (Demo + Pro).

### 5. Jalankan backend

```bash
cd backend
make run
# listening on :8080
```

### 6. Jalankan frontend

```bash
cd frontend
bun install
bun run dev
# http://localhost:3000
```

---

## Penggunaan

1. Buka `http://localhost:3000` dan login sebagai super admin.
2. Buat organisasi baru via menu Admin → masukkan nama, slug, dan email org admin.
3. Login sebagai org admin, ganti password sementara.
4. Upload dokumen hukum (PDF teks, DOCX, atau TXT, maks 20 MB) di halaman **Pustaka**.
5. Tunggu status berubah menjadi **Terindeks**, lalu buka halaman **Chat** dan mulai bertanya.

> PDF hasil scan (tanpa teks selectable) tidak didukung — dokumen akan berstatus `Gagal` dengan keterangan jelas.

---

## Perintah backend

```bash
make run          # jalankan server
make migrate-up   # terapkan semua migrasi
make migrate-down # rollback satu migrasi
make seed         # seed super admin + plans
make test         # jalankan semua unit test
make gen          # regenerasi DTO dari openapi.yaml
make tidy         # go mod tidy
```

---

## Struktur direktori

```
backend/
  cmd/server/         entrypoint
  config/             konfigurasi dari env
  db/migrations/      file SQL migrasi
  db/seed/            seed super admin + plans
  internal/
    domain/           tipe domain + interface repo
    repository/       implementasi postgres + qdrant
    usecase/          logika bisnis
    delivery/http/    handler, router, middleware, DTO
  pkg/                utilitas (jwt, hash, embedding, llm, chunk, extract, storage)

frontend/
  app/                halaman Next.js (App Router)
  components/         komponen UI
  lib/                api client, auth context
```

---

## Catatan deployment

- Ganti `JWT_SECRET` dengan nilai acak yang kuat sebelum deploy.
- Set `COOKIE_SECURE=true` jika menggunakan HTTPS.
- Isi `CORS_ORIGINS` dengan domain frontend produksi.
- Model chat dan embedding di-set lewat `.env` (`CHAT_MODEL`, `EMBEDDING_MODEL`). Tidak semua model di Maia Router aktif; sesuaikan dengan katalog yang tersedia.
