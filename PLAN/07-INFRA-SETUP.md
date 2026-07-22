# 07 — Infra Setup (localhost)

Acuan konkret Fase 0. **Infra (Postgres, Qdrant) lewat docker-compose. App (backend, frontend) jalan native** biar hot reload enak.

- Backend: `make run` (di `backend/`)
- Frontend: `bun run dev` (di `frontend/`)
- Infra: `docker compose up -d`

**Tidak ada cloud, domain, atau TLS.**

## 1. Port map

| Komponen | Jalan via | Host port | Akses |
|----------|-----------|-----------|-------|
| Postgres | docker | `5432` | `localhost:5432` |
| Qdrant | docker | `6333` (HTTP), `6334` (gRPC) | dashboard `localhost:6333/dashboard` |
| Backend (Go) | `make run` (native) | `8080` | `localhost:8080`, docs `/docs` |
| Frontend (Next) | `bun run dev` (native) | `3000` | `localhost:3000` (+ `app.localhost`, `admin.localhost`) |

> n8n tidak dipakai di MVP. Infra cukup postgres + qdrant.

## 2. `docker-compose.yml` (infra saja)

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports: ["5432:5432"]
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 5s
      timeout: 5s
      retries: 10

  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333", "6334:6334"]
    volumes: ["qdrantdata:/qdrant/storage"]

volumes:
  pgdata:
  qdrantdata:
```

## 3. `.env.example`

```dotenv
# Postgres
POSTGRES_USER=applaw
POSTGRES_PASSWORD=change_me_locally
POSTGRES_DB=applaw

# Backend
PORT=8080
DATABASE_URL=postgres://applaw:change_me_locally@localhost:5432/applaw?sslmode=disable
QDRANT_URL=http://localhost:6333

# Auth
JWT_SECRET=dev_secret_change_me
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# AI
ANTHROPIC_API_KEY=sk-ant-...
CHAT_MODEL=[REDACTED]

# Embedding — LOKAL via Ollama (diputuskan)
EMBEDDING_PROVIDER=local
EMBEDDING_URL=http://localhost:11434    # ollama native di host
EMBEDDING_MODEL=nomic-embed-text
EMBEDDING_DIM=768                 # ukuran vektor Qdrant, ikut model

# RAG tuning
RAG_TOP_K=5
CHUNK_SIZE=800
CHUNK_OVERLAP=100

# Storage
STORAGE_DIR=./storage             # file upload di disk lokal

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8080
```

`.env` di-gitignore. Commit `.env.example` saja.

## 4. Urutan bring-up

```bash
# prasyarat: Ollama terinstall + model embedding ditarik (native, bukan docker)
ollama pull nomic-embed-text       # butuh untuk ingestion & RAG (Fase 2+)
# prasyarat: poppler-utils (pdftotext) + tesseract-ocr terinstall untuk Fase 2

cp .env.example .env               # isi ANTHROPIC_API_KEY
docker compose up -d               # postgres, qdrant
cd backend && make migrate-up && make run  # backend native :8080
cd frontend && bun install && bun run dev  # frontend native :3000
```

> Ollama, poppler, tesseract = tool host (native), bukan service compose. Fase 0 belum butuh ketiganya; wajib ada sebelum Fase 2.

## 5. Verifikasi (definition of done Fase 0)

| Komponen | Cek | Sehat kalau |
|----------|-----|-------------|
| Postgres | `docker compose exec postgres pg_isready -U applaw` | `accepting connections` |
| Qdrant | `curl localhost:6333/healthz` | `200` |
| Backend | `curl localhost:8080/healthz` | `{"status":"ok"}` |
| Frontend | buka `localhost:3000` | landing (dari client) render |

Keempat sehat → Fase 0 selesai.

## 6. Yang SENGAJA belum dibuat (localhost)

- Reverse proxy / Nginx / Traefik — akses langsung port.
- TLS/HTTPS — localhost.
- Dockerize backend/frontend untuk dev — jalan native (Dockerfile ada tapi buat deploy nanti).
- Object storage (S3) — file di disk lokal dulu (`pkg/storage` local impl).
- Secret manager — cukup `.env`.
- **n8n** — ingestion MVP pakai Go goroutine. n8n masuk saat butuh scheduled workflow eksternal (update besar #1).
- Config Coolify.

Tambah saat pindah ke server. Bukan sekarang.
