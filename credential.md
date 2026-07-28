# MindLaw / Lexora — Kredensial & Data Deploy

> **RAHASIA.** Repo ini privat. Jangan pindahkan file ini ke repo publik, jangan share ke luar tim.
> Terakhir diperbarui: 2026-07-29

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
| Host | `postgres:5432` |
| DB name | `mindlaw` |
| User | `mindlaw` |
| Password | `ricZ0i1xOQNMc5bPiqXSQFAB7p9F7AioOvvnVujm` |

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
