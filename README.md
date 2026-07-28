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
  cmd/usertool/       CLI CRUD akun (dipakai make user-*)
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

## Deploy Produksi

Produksi jalan di VPS (`103.193.179.9`, domain `mindlaw.web.id`) sebagai stack Docker: Postgres, Qdrant, backend Go, frontend Next.js, dan Caddy (reverse proxy + HTTPS otomatis). Semua dikendalikan lewat `Makefile` di root dari laptop, via SSH key `~/.ssh/mindlaw_deploy`.

> Kredensial lengkap (server, akun, DB, API key) ada di [`credential.md`](credential.md).

### Deploy

```bash
make deploy            # sync kode + build + migrate + seed + up (full, aman diulang)
make deploy-backend    # rebuild + restart backend saja (cepat)
make deploy-frontend   # rebuild + restart frontend saja
```

`make deploy` mengirim kode via `tar` over SSH ke `/root/lexora`, lalu menjalankan `deploy.sh` (idempotent: migrate → seed → up). File secret (`.env`, `.env.production`, dll.) **tidak** ikut ter-sync, jadi konfigurasi di server tidak tertimpa.

### Operasional

```bash
make ps                # status container
make logs S=backend    # ikuti log (S kosong = semua service)
make restart S=frontend
make shell             # SSH masuk ke server
make help              # daftar lengkap perintah
```

### CRUD akun (dari laptop, tanpa masuk server)

```bash
make user-list
make user-create EMAIL=a@b.com PASS=xxx NAME="Nama" [ROLE=none|super_admin]
make user-passwd EMAIL=a@b.com PASS=baru
make user-activate EMAIL=a@b.com
make user-deactivate EMAIL=a@b.com
make user-delete EMAIL=a@b.com
```

Perintah `user-*` menjalankan `backend/cmd/usertool` di dalam container backend, langsung ke database produksi.

### Checklist sebelum deploy pertama

- `JWT_SECRET` & `JWT_ADMIN_SECRET` diisi nilai acak kuat (bukan nilai dev).
- `COOKIE_SECURE=true` (wajib di HTTPS; kalau `false`, login prod gagal diam-diam).
- `CORS_ORIGINS_APP` / `CORS_ORIGINS_ADMIN` diisi domain frontend produksi (tanpa wildcard).
- Model chat & embedding sesuai katalog Maia Router yang aktif.
- DNS `mindlaw.web.id` + subdomain `app.` `admin.` `api.` mengarah ke IP server.
