# MindLaw / Lexora — Kredensial & Data Deploy

> **RAHASIA.** Repo ini privat. Jangan pindahkan file ini ke repo publik, jangan share ke luar tim.
> Terakhir diperbarui: 2026-07-30 (+ env Update 6)

---

## URL Publik

| Layanan | URL |
|---|---|
| Landing / app | https://mindlaw.web.id |
| App user | https://app.mindlaw.web.id |
| Admin panel | https://admin.mindlaw.web.id |
| API backend | https://api.mindlaw.web.id (health: `/healthz`) |

---

## Server (VPS)

| Item | Nilai |
|---|---|
| IP publik | `103.193.179.9` |
| Hostname | `srv1.mindlaw.web.id` |
| User SSH | `root` |
| Password root | `lB9xXFoFSGjAYbwZId` |
| SSH key deploy | `~/.ssh/mindlaw_deploy` (tanpa password, dipakai `make`) |
| Lokasi project | `/root/lexora` |
| Spec | 8GB RAM, Ubuntu 22.04, ~97G disk, +2GB swap |

> Catatan: `.env.deploy` lama mencatat password root berbeda (`+.Mpf4f8PqpS{c6@`) untuk IP yang sama. Yang valid/dipakai saat ini adalah `lB9xXFoFSGjAYbwZId`.

---

## Akun Aplikasi

### Super Admin (login di admin.mindlaw.web.id)

| Field | Nilai |
|---|---|
| Email | `superadmin@mindlaw.web.id` |
| Password | `mL9vX2qN8wKpR5tY3hJdF7cB4sAeG6uZ` |

### Akun seed (dev/demo — password default `password123`)

| Peran | Email | Org |
|---|---|---|
| Org Admin | `admin@mindlaw.web.id` | Firma Hukum MindLaw |
| User Pro | `pro@mindlaw.web.id` | Firma Hukum MindLaw |
| User Free | `free@mindlaw.web.id` | Kantor Hukum Merdeka |

> Saran: ganti password akun seed di produksi — `make user-passwd EMAIL=admin@mindlaw.web.id PASS=...`

---

## Database (Postgres — internal docker network, tidak diekspos publik)

| Field | Nilai |
|---|---|
| Host | `postgres:5432` (dari host: `localhost:5432`) |
| DB name | `applaw` |
| User | `applaw` |
| Password | `change_me_locally` |

> Creds di-set saat volume Postgres pertama dibuat (deploy awal) dan gak berubah walau `.env` diedit. Nilai di atas = kondisi asli server saat ini. Cek langsung: `psql -U applaw -d applaw`.
> Password masih placeholder tapi aman-aman aja: Postgres gak diekspos publik, cuma diakses backend lewat docker network internal.

---

## Secret Auth (JWT)

| Field | Nilai |
|---|---|
| `JWT_SECRET` | `r2y5ym8UCghJ9ZzpK16l1NDRt47mBwr4nbfC7pB9` |
| `JWT_ADMIN_SECRET` | `yewrJ07xTd5MsvCSvY1FUGXOthwx6aiJQJzlzOby` |
| Access TTL | `15m` |
| Refresh TTL | `720h` (30 hari) |

---

## AI / Maia Router

| Field | Nilai |
|---|---|
| `MAIA_API_KEY` | `sk-5HsuYGO50GQ2bM59_n2Qig` |
| Chat (High) | `maia/claude-sonnet-4-5` |
| Chat (Normal) | `anthropic/claude-haiku-4-5` |
| Web search | `openai/gpt-4o-mini-search-preview` |
| Embedding | `openai/text-embedding-3-large` (dim 3072) |
| Embedding URL | `https://api.maiarouter.ai/v1` |

---

## Update 6 — Env baru (isi saat kredensial turun)

Semua **opsional**: kosong = fitur mati, sistem tetap jalan (mode lama). Set di `.env.production` server.

### Mailer (Gmail SMTP) — register verif + alert saldo Maia
| Field | Nilai | Catatan |
|---|---|---|
| `SMTP_USER` | `mindlaw.env@gmail.com` | akun pengirim |
| `SMTP_APP_PASSWORD` | *(App Password Google — TODO)* | butuh 2FA aktif dulu, lalu generate |
| `APP_BASE_URL` | `https://app.mindlaw.web.id` | untuk link verifikasi email |

> Kosong = register langsung aktif tanpa verifikasi email (dev). Alert Maia mati.

### Login Google (OAuth Client ID — bukan rahasia)
| Field | Nilai | Catatan |
|---|---|---|
| `GOOGLE_CLIENT_ID` | *(dari Google Cloud Console — TODO)* | backend verif `aud` |
| `NEXT_PUBLIC_GOOGLE_CLIENT_ID` | *(sama dgn atas)* | FE `.env`, render tombol GIS |

> Kosong = tombol Google disembunyikan / endpoint `/auth/google` tolak.

### Alert saldo Maia
| Field | Nilai | Catatan |
|---|---|---|
| `MAIA_BALANCE_THRESHOLD` | `20` | USD; 0 = ticker mati |
| `MAIA_TOPUP_TOTAL_USD` | *(total top-up manual Maia)* | basis estimasi |

> ⚠️ Harga per-token di [pkg/pricing/pricing.go](../backend/pkg/pricing/pricing.go) masih **placeholder** (list-price Anthropic). Ganti dgn harga efektif Maia sebelum andalkan angka estimasi.

### Xendit (payment gateway)
| Field | Nilai | Catatan |
|---|---|---|
| `XENDIT_SECRET_KEY` | *(dari dashboard Xendit — TODO KYC)* | rahasia server-side |
| `XENDIT_CALLBACK_TOKEN` | *(set di dashboard webhook)* | verifikasi webhook |
| `XENDIT_SUCCESS_URL` | `https://app.mindlaw.web.id/app/billing?paid=1` | redirect after-pay |

> Secret kosong = mode manual (invoice pending tanpa checkout URL, mark-paid manual super_admin).

### Runbook deploy Update 6
1. Migrasi `000013` jalan otomatis via `make deploy` (atau `migrate ... up`).
2. **Seed org internal** sekali: `make user-create` / buat org via panel super_admin dgn Name "Mind Law Internal", Slug `mindlaw` (dropdown Assign Akun default cari slug ini).
3. `CORS_ORIGINS_APP` **wajib** muat `https://mindlaw.web.id,https://app.mindlaw.web.id` (sudah ada di `.env.production`).
4. Uji Xendit di **test mode** dulu → flip live key setelah KYC lolos.

---

## Command Cepat (dari laptop, di folder project)

```bash
# Deploy
make deploy            # sync + build + migrate + seed + up (full)
make deploy-backend    # rebuild backend saja
make deploy-frontend   # rebuild frontend saja

# Ops
make ps                # status container
make logs S=backend    # ikuti log (S kosong = semua)
make restart S=frontend
make shell             # SSH masuk server
make help

# CRUD akun
make user-list
make user-create EMAIL=a@b.com PASS=xxx NAME="Nama" [ROLE=none|super_admin]
make user-passwd EMAIL=a@b.com PASS=baru
make user-activate EMAIL=a@b.com
make user-deactivate EMAIL=a@b.com
make user-delete EMAIL=a@b.com
```
