# 12 — Papan Tanya-Jawab (Lexora)

Tempat aku (Claude) naruh pertanyaan buat kamu, dan rekam keputusan. Alur:
**OPEN** = nunggu jawaban kamu · **ANSWERED** = udah dijawab, keputusan dicatat.

Jawab dengan nulis di bawah pertanyaan. Kalau udah masuk PLAN, aku pindah ke arsip.

---

## OPEN — perlu jawaban

_(semua pertanyaan sudah dijawab)_

---

## ANSWERED — arsip keputusan

| # | Topik | Keputusan | Di PLAN |
|---|-------|-----------|---------|
| 1 | Embedding | Lokal `nomic-embed-text` dim 768; fallback Voyage kalau berat | 02 §7 |
| 2 | Landing | Masuk MVP. Nama project: **Lexora** | 01, 09 |
| 3 | Copy landing | Aku draft ID; EN template ada → bilingual bonus | 09 §8 |
| 4 | Timeline | Santai, no deadline | — |
| 5 | Internasional | MVP fokus ID; seam didokumentasi, belum dibangun | 02 §9, 03 |
| 6 | Tier & model | Fix Claude. Demo=Haiku, Pro=Sonnet ($17/seat), Premium=Opus (nanti) | 02 §7b, 03 |
| S | Security | 12 ancaman dipetakan; basic dijaga dari awal | 10 |
| 7 | Domain | `lexora.id` (TLD diasumsikan; konfirmasi sebelum config prod) | 04, 10 |
| 8 | Bahasa jawaban AI | Ikut bahasa pertanyaan; default ID kalau ambigu | 08 §2 |
| 9 | Dokumen contoh RAG | Client sediakan 2-3 PDF hukum saat Fase 2 | 05 §Fase 2 |
| 10 | Ingestion | Go worker goroutine (bukan n8n). n8n = rencana update besar #1 | 02 §3, §10 |
| 11 | Migration tool | `golang-migrate`, `make migrate-up`/`migrate-down` | 08 §2 |
| 12 | Embedding server | Ollama native (`nomic-embed-text`, :11434) | 02 §7a |
| 13 | Ekstraksi teks | `pdftotext` + fallback OCR Tesseract (scan) | 02 §7a |
| 14 | Chunking | Fixed ~800 token, overlap 100 (default, tunable) | 02 §7a |
| 15 | No-match RAG | AI jawab jujur tanpa citation palsu, jangan halusinasi | 06 §2, 01 §4.5 |
| 16 | Refresh token | Tabel `refresh_tokens` (hash + rotation + revoke) | 03 ERD |
| 17 | Billing model | Subscription per org + seats (bukan per user) | 03 ERD, 01 §5 |
| 18 | Manajemen user MVP | Admin buat akun + password sementara; email/reset = fase 2 | 01 §5 |
| 19 | Export | PDF chromedp, Word unioffice | 02 §7c |
| 20 | SSE di FE | `fetch`+ReadableStream (bukan EventSource) | 06 §6 |
| 21 | RBAC | Matriks 3 role × aksi × scope. Org Admin & Super Admin **tidak** baca isi chat/dokumen user (metadata saja) | 10 §Matriks RBAC, 01 §3 |

## Catatan kecil (default dipakai, koreksi kalau salah)

- Demo tier: Claude Haiku.
- Fallback Voyage: kalau pindah, `EMBEDDING_DIM` berubah → re-index. Kabarin sebelum dokumen numpuk.
- Bilingual landing: ID dulu, toggle EN kalau sempat.
- Domain: `lexora.id` diasumsikan. Konfirmasi TLD sebelum config prod CORS/email.
