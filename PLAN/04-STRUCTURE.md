# 04 — Struktur Repo

Monorepo. **`backend/`** (Go Clean Architecture) + **`frontend/`** (Next, dari client) + **OpenAPI kontrak** + infra compose.

> **`frontend/` sudah ada** (dikirim client). **Jangan rebuild.** Halaman app baru = tambah route di `frontend/app/`.
> **Kontrak API = OpenAPI** (`openapi/openapi.yaml`). Spec-first: backend generate server interface + frontend generate typed client dari file yang sama.
> **Run:** backend `make run`, frontend `bun run dev`. Infra (postgres + qdrant) lewat docker-compose.

## 1. Top-level

```
project_AI_law/
├── docker-compose.yml          # infra saja: postgres, qdrant
├── .env.example
├── openapi/
│   └── openapi.yaml            # KONTRAK API (OpenAPI 3.1) — dipakai backend & frontend
├── PLAN/                        # dokumen perancangan
├── backend/                     # Go (Clean Architecture) — §2
├── frontend/                    # Next 16 (dari client) — §4
└── n8n/
    └── workflows/               # placeholder — diisi saat update besar #1 (ingestion eksternal terjadwal)
```

## 2. Backend (Go — Clean Architecture)

Aturan dependensi **menunjuk ke dalam**: `delivery → usecase → domain ← repository`.
Domain nggak tahu soal HTTP/Postgres/Qdrant. Layer luar implement interface yang dideklarasi domain.

```
backend/
├── Makefile                        # make run, make migrate, make gen, make test
├── cmd/
│   └── server/
│       └── main.go                 # wiring/DI: config → repo → usecase → handler → server
├── config/
│   └── config.go                   # load env
│
├── internal/
│   ├── domain/                     # LAYER 1: entity + kontrak interface (no deps)
│   │   ├── user.go                 #   User + UserRepository + UserUsecase
│   │   ├── organization.go
│   │   ├── membership.go
│   │   ├── workspace.go
│   │   ├── chat.go
│   │   ├── message.go
│   │   ├── document.go             #   Document + DocumentRepository + VectorRepository
│   │   ├── citation.go
│   │   ├── subscription.go
│   │   ├── plan.go
│   │   ├── prompt.go
│   │   ├── aimodel.go
│   │   ├── audit.go
│   │   └── errors.go
│   │
│   ├── usecase/                    # LAYER 2: business logic (depend interface domain saja)
│   │   ├── auth_usecase.go
│   │   ├── organization_usecase.go
│   │   ├── workspace_usecase.go
│   │   ├── chat_usecase.go
│   │   ├── rag_usecase.go          #   embed → search Qdrant → LLM → citation
│   │   ├── document_usecase.go     #   simpan file → spawn goroutine ingestion
│   │   ├── billing_usecase.go      #   cek limit token
│   │   ├── prompt_usecase.go
│   │   ├── dashboard_usecase.go
│   │   └── audit_usecase.go
│   │
│   ├── repository/                 # LAYER 3: impl kontrak domain (adapter data)
│   │   ├── postgres/
│   │   │   ├── db.go               #   pgx pool + tx helper
│   │   │   ├── user_repo.go
│   │   │   ├── organization_repo.go
│   │   │   ├── chat_repo.go
│   │   │   ├── document_repo.go
│   │   │   ├── subscription_repo.go
│   │   │   └── ...                 #   satu file per entity
│   │   └── qdrant/
│   │       └── vector_repo.go      #   impl VectorRepository
│   │
│   └── delivery/                   # LAYER 4: transport (depend interface usecase)
│       └── http/
│           ├── server.go           #   http.Server + graceful shutdown
│           ├── router.go           #   daftar route + pasang middleware
│           ├── handler/            #   tipis: parse req → usecase → respond
│           │   ├── auth_handler.go
│           │   ├── organization_handler.go
│           │   ├── workspace_handler.go
│           │   ├── chat_handler.go     #   termasuk streaming SSE
│           │   ├── document_handler.go
│           │   ├── billing_handler.go
│           │   ├── prompt_handler.go
│           │   ├── dashboard_handler.go
│           │   ├── webhook_handler.go  #   webhook internal (future: n8n fase 2)
│           │   └── health_handler.go
│           ├── middleware/
│           │   ├── auth.go          #   verify jwt
│           │   ├── tenant.go        #   inject org id
│           │   ├── rbac.go          #   guard role
│           │   ├── logger.go
│           │   ├── recover.go
│           │   ├── cors.go          #   izinkan origin subdomain
│           │   └── ratelimit.go
│           └── dto/
│               └── gen.go           #   hasil oapi-codegen (types + server interface)
│
├── pkg/                            # util lintas-domain, tanpa business logic
│   ├── jwt/
│   ├── hash/                       # argon2id / bcrypt
│   ├── llm/                        # Claude client (model param: haiku/sonnet/opus)
│   ├── embedding/                  # interface Embedder + impl lokal/voyage (02 §7)
│   ├── storage/                    # file storage (local → s3)
│   ├── logger/                     # structured json
│   └── validator/
│
├── db/
│   ├── migrations/                 # SQL eksplisit (golang-migrate/goose)
│   └── seed/                       # seed super admin, plan default
├── go.mod
└── Dockerfile                      # buat deploy nanti, bukan dev
```

**Run backend:** `make run` (di dalam `backend/`). Target lain: `make migrate`, `make gen` (oapi-codegen), `make test`.

### Aturan Clean Architecture (wajib)

1. **Domain nggak import** apa-apa dari layer luar. Cuma entity + interface + error.
2. **Usecase depend interface domain**, bukan struct konkret. Testable tanpa DB (mock).
3. **Repository implement interface domain.** Detail Postgres/Qdrant terkurung di sini.
4. **Handler tipis.** Decode → usecase → encode. Nggak ada SQL/logika bisnis di handler.
5. **Wiring di `cmd/server/main.go`.** DI manual, jangan framework DI berat.
6. **Isolasi tenant** di `middleware/tenant.go` + enforce di usecase/repo. Invariant keamanan.

## 3. OpenAPI — kontrak API (spec-first)

- **Sumber kebenaran:** `openapi/openapi.yaml` (OpenAPI 3.1).
- **Backend:** `oapi-codegen` → types + server interface di `internal/delivery/http/dto/gen.go`. Handler implement interface itu. Spec berubah → compile error nunjuk yang belum update. Jalanin via `make gen`.
- **Frontend:** `openapi-typescript` (atau orval) → typed client di `frontend/lib/api/`. Frontend nggak nulis tipe manual.
- **Docs:** Swagger UI di `localhost:8080/docs` (dev).
- Alur ganti endpoint: **edit `openapi.yaml` dulu** → regen backend & frontend → implement.

## 4. Frontend (Next + Subdomain)

Satu app Next, **route berdasar subdomain** via `middleware.ts`. Satu app, bukan multi-deploy.
**Package manager & runner = bun.**

```
frontend/
├── middleware.ts                   # host → zona rewrite
├── app/
│   ├── layout.tsx                  # root: font Inter+Playfair, globals (lang id)
│   ├── globals.css                 # design tokens (jangan diacak)
│   │
│   ├── (marketing)/                # ZONA root (www) — SUDAH ADA, lokalisasi ke ID
│   │   ├── page.tsx                #   landing
│   │   └── platform/page.tsx
│   │
│   ├── (auth)/                     # login/register (dipakai app & admin)
│   │   └── login/page.tsx
│   │
│   ├── app/                        # ZONA app.<domain> — dibangun (user app)
│   │   ├── layout.tsx              #   layout sidebar
│   │   ├── dashboard/page.tsx
│   │   ├── chat/[id]/page.tsx      #   streaming + citation
│   │   ├── knowledge-base/page.tsx
│   │   ├── workspaces/page.tsx
│   │   └── settings/page.tsx
│   │
│   └── admin/                      # ZONA admin.<domain> — dibangun (super admin)
│       ├── layout.tsx
│       ├── users/page.tsx
│       ├── organizations/page.tsx
│       ├── ai-models/page.tsx
│       ├── prompts/page.tsx
│       ├── knowledge-base/page.tsx
│       └── billing/page.tsx
│
├── components/
│   ├── ui/                         # shadcn primitives (sudah ada)
│   ├── marketing/                  # hero, capabilities, dll (sudah ada)
│   ├── app/                        # sidebar, chat-view, citation-card, token-meter
│   └── admin/
├── lib/
│   ├── api/                        # typed client (generated dari openapi.yaml)
│   ├── auth/                       # token/session helper
│   └── utils.ts                    # sudah ada
├── middleware.ts
├── components.json                 # shadcn base-nova
└── package.json                    # bun, Next 16.2, React 19
```

**Run frontend:** `bun run dev` (di dalam `frontend/`). Install: `bun install`. Tambah komponen: `bunx shadcn@latest add <x>`.

> Client kirim dengan `pnpm-lock.yaml`. Karena pindah ke bun: hapus `pnpm-lock.yaml`, jalanin `bun install` sekali (bikin `bun.lock`).

### Subdomain map

| Produksi (nanti) | Dev (localhost) | Zona | Isi |
|------------------|-----------------|------|-----|
| `lexora.id` | `localhost:3000` | `(marketing)` | Landing, `/platform` |
| `app.lexora.id` | `app.localhost:3000` | `app/` | Dashboard, chat, KB, workspace |
| `admin.lexora.id` | `admin.localhost:3000` | `admin/` | Panel super admin |

- `*.localhost` otomatis resolve ke 127.0.0.1 di Chrome/Firefox — subdomain jalan di dev tanpa edit hosts.
- Org Admin **tidak** dapat subdomain sendiri di MVP — masuk lewat `app.` dengan role `org_admin` (menu kondisional). Subdomain per-org = fase lanjutan.

### `middleware.ts` (inti)

```ts
// one app, subdomain rewrite
export function middleware(req: NextRequest) {
  const host = req.headers.get('host') ?? ''
  const sub = host.split('.')[0]            // subdomain
  const url = req.nextUrl
  if (sub === 'admin') url.pathname = `/admin${url.pathname}`
  else if (sub === 'app') url.pathname = `/app${url.pathname}`
  // else marketing
  return NextResponse.rewrite(url)
}
```

Middleware juga blok akses `/app` & `/admin` langsung dari host marketing.

## 5. Prinsip

- **Backend = Clean Architecture** (§2). Dependensi ke dalam, handler tipis, logika di usecase.
- **OpenAPI spec-first** (§3). Edit spec → regen → implement.
- **Migration SQL di-track** (`db/migrations`), bukan auto-migrate ORM.
- **Frontend:** landing sudah ada → lokalisasi, jangan rebuild. Subdomain via `middleware.ts`. Semua UI pakai token design system ([09](09-DESIGN-SYSTEM.md)).
- **bun** untuk frontend, `make run` untuk backend.
- **Workflow n8n** — folder `n8n/workflows/` sudah ada sebagai placeholder. Diisi saat update besar #1 (ingestion sumber eksternal terjadwal).

> Belum bikin file yang belum kepakai. Tapi kerangka lapisan (domain/usecase/repository/delivery) dipasang dari awal biar konsisten.
