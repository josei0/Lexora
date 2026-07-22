# PLAN — Lexora (AI Legal Assistant Platform)

Dokumen perancangan. Baca urut.

| # | Dokumen | Isi |
|---|---------|-----|
| 01 | [PRD](01-PRD.md) | Product Requirements — tujuan, user, fitur, scope MVP vs fase 2 |
| 02 | [Tech Stack](02-TECH-STACK.md) | Stack, arsitektur, Go worker ingestion, model AI, deploy |
| 03 | [ERD](03-ERD.md) | Skema database (diagram + kamus tabel) |
| 04 | [Structure](04-STRUCTURE.md) | Struktur folder repo (Go BE + Next FE) |
| 05 | [Planning](05-PLANNING.md) | Roadmap bertahap, milestone |
| 06 | [Flows](06-FLOWS.md) | Alur utama: RAG, ingestion, auth |
| 07 | [Infra Setup](07-INFRA-SETUP.md) | **Konkret**: docker-compose, .env, port, verifikasi (localhost) |
| 08 | [Agent Brief](08-AGENT-BRIEF.md) | **Baca dulu** kalau kamu agent eksekutor — konvensi, DoD, guardrail |
| 09 | [Design System](09-DESIGN-SYSTEM.md) | Token warna/font/komponen dari FE client — acuan semua UI |
| 10 | [Security](10-SECURITY.md) | 12 ancaman (SQLi, XSS, CSRF, IDOR, SSRF, JWT, dll) → mitigasi per layer |
| 11 | [External Sources](11-EXTERNAL-SOURCES.md) | Sumber hukum eksternal (JDIHN/BPK/Hukumonline) — fase 2 |
| 12 | [Questions](12-QUESTIONS.md) | Papan tanya-jawab: pertanyaan terbuka + arsip keputusan |

**Status:** Draft v0.4 · Sumber hukum diarsipkan, papan tanya-jawab aktif di 12.

> **Untuk agent eksekutor (Claude session lain):** mulai dari **[08-AGENT-BRIEF](08-AGENT-BRIEF.md)**, lalu ikuti fase di 05-PLANNING. Deploy **localhost only** (backend `make run`, frontend `bun run dev`, infra docker-compose). Landing sudah ada dari client (folder `frontend/`) — **jangan rebuild**, lokalisasi ke ID + bangun halaman app pakai [09-DESIGN-SYSTEM](09-DESIGN-SYSTEM.md).

## Keputusan kunci (ringkas)

- **Produk:** Lexora. **MVP first.** Fitur hukum lanjutan (kronologi, draft template, sumber eksternal) = fase 2.
- **Stack:** Next **16** (frontend dari client, **bun**) · Go **Clean Architecture** (`make run`) · PostgreSQL · Qdrant · Claude · **OpenAPI** kontrak.
- **Struktur:** `backend/` + `frontend/` + `openapi/`. Subdomain `admin.`/`app.`/root (lihat 04).
- **Frontend:** landing dari client (folder `frontend/`, template — **lokalisasi ke ID**). App (dashboard/chat/KB/workspace) **dibangun** pakai design system client.
- **Embedding:** lokal (`nomic-embed-text`, dim 768) — diputuskan.
- **Design system:** navy + emas + cream, Inter + Playfair — acuan semua UI (lihat 09).
- **Multi-tenant:** shared DB + `organization_id`. 1 user = 1 org (via tabel `memberships`, siap many-to-many).
- **n8n** = rencana update besar #1 (ingestion sumber eksternal terjadwal). **Go goroutine** = ingestion MVP.
- **Billing:** per user (seat), 2 tier — Demo (gratis, model murah) / Pro ($17/seat, Claude Sonnet). Model beda per tier. Soft→hard limit.
- **Security:** basic dijaga dari awal (SQLi/XSS/CSRF/IDOR/JWT/CORS/dll) — lihat [10-SECURITY](10-SECURITY.md).
- **Deploy:** **docker-compose di localhost** (Coolify nanti setelah MVP).
- **Arah masa depan:** 2 update besar — (1) sekarang **hukum Indonesia**, (2) nanti **internasional**. MVP fokus ID; titik ekstensi didokumentasikan, belum dibangun.
- **Terbuka:** provider embedding (lokal vs Voyage) — kunci sebelum Fase 2. Lihat `question.md`.
