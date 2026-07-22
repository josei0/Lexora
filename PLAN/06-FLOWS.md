# 06 — Flows Utama

## 1. Ingestion Knowledge Base (async, Go background goroutine)

```mermaid
sequenceDiagram
    actor Admin
    participant Web as Next.js
    participant API as Go API
    participant S as Storage
    participant W as Go Worker (goroutine)
    participant Q as Qdrant
    participant DB as Postgres

    Admin->>Web: Upload PDF/DOCX/TXT
    Web->>API: POST /documents
    API->>API: validasi MIME (sniff isi) + ukuran ≤20MB
    API->>S: simpan file ke disk
    API->>DB: insert documents (status=uploaded)
    API-->>Web: 202 Accepted
    API->>W: kirim ke worker pool (channel)
    W->>DB: set status=processing
    W->>Q: ensure collection kb_<org_id> (idempotent)
    W->>S: baca file
    W->>W: ekstrak teks (pdftotext, fallback OCR) → Chunking → Embedding
    W->>Q: upsert vectors (payload: metadata citation)
    W->>DB: insert document_chunks, update status=indexed
```

### Durability & konkurensi (penting — pengganti n8n)

- **Worker pool terbatas**, bukan goroutine tak terbatas. Upload masuk channel; N worker (mis. 3) ambil satu-satu. Cegah OOM saat banyak PDF 20MB barengan.
- **Recovery saat startup:** waktu backend start, scan `documents WHERE status IN ('uploaded','processing')` → re-enqueue. Ini yang bikin goroutine aman walau server restart (kelemahan utama drop n8n → ditutup di sini).
- **Gagal = `status=failed`** + catat alasan (kolom `error` opsional / audit log). Jangan diam. User bisa retry dari UI.
- **Collection Qdrant dibuat lazy & idempotent:** `ensure kb_<org_id>` dipanggil sebelum upsert pertama. Tidak dibuat saat org dibuat (org bisa tak punya dokumen).

## 2. Chat + RAG (realtime, di Go)

```mermaid
sequenceDiagram
    actor User
    participant Web as Next.js
    participant API as Go API
    participant Q as Qdrant
    participant CL as Claude

    User->>Web: "Bagaimana syarat PKPU?"
    Web->>API: POST /chats/{id}/messages (SSE)
    API->>API: cek limit token (soft/hard)
    API->>CL: embed pertanyaan
    API->>Q: search top-k (filter scope + org_id)
    Q-->>API: chunks + skor
    API->>API: filter skor ≥ threshold
    alt ada chunk relevan
        API->>CL: prompt(system + konteks + pertanyaan) [stream]
        CL-->>API: jawaban (stream)
        API-->>Web: stream token + citations
    else tidak ada match
        API-->>Web: "Tidak ditemukan dasar hukum di basis pengetahuan." (tanpa citation palsu)
    end
    API->>DB: simpan message + citations + token_usage
```

### Perilaku "no-match" (resolusi konflik citation-wajib)

Aturan PRD: tiap jawaban wajib bersumber. Maka **kalau tidak ada chunk relevan (skor < threshold, atau KB kosong), AI TIDAK mengarang jawaban.** Balas jujur: "Tidak ditemukan dasar hukum yang relevan di basis pengetahuan untuk pertanyaan ini." System prompt harus melarang jawab dari pengetahuan umum tanpa sumber KB. Ini mencegah halusinasi hukum — lebih bahaya daripada tidak menjawab.

### Format SSE (kontrak FE ↔ BE)

Endpoint streaming pakai 3 event type. FE dan BE **wajib** konsisten:

```
data: {"token":"Syarat"}          <- per token, berkali-kali
data: {"token":" PKPU"}
data: {"done":true,"citations":[{"label":"UU No.37/2004 Pasal 222","document_id":"...","chunk_id":"...","page_no":12}]}
data: {"error":"limit token tercapai"}   <- kalau gagal, ganti done
```

- Tiap baris `data:` = satu JSON, dipisah `\n\n` (spec SSE).
- Event `done` selalu terakhir saat sukses, berisi daftar citation lengkap.
- Kalau error di tengah stream, kirim event `error` lalu tutup koneksi.
- Header: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`.

## 3. Auth & Tenant Scoping

```mermaid
sequenceDiagram
    actor User
    participant API as Go API
    participant DB as Postgres

    User->>API: POST /login {email, password}
    API->>DB: cari user, verify hash
    API-->>User: access JWT (+refresh) berisi user_id, org_id, role
    User->>API: request + Bearer JWT
    API->>API: middleware: extract org_id + role
    API->>DB: query di-scope WHERE organization_id = ?
    API-->>User: data milik org itu saja
```

## 4. Limit Token

**Hitung token pakai Claude token counting endpoint** (`POST /v1/messages/count_tokens`), bukan estimasi karakter — biar akurat.

Pemakaian dihitung dari `SUM(token_usage)` user di periode berjalan (bukan counter — hindari drift). Limit per user = `plans.monthly_token_limit` (jatah per seat). Query di-index `token_usage(user_id, created_at)`.

- **Sebelum kirim ke Claude:** hitung input token via `count_tokens`. Ambil `used = SUM(input+output)` user ini sejak `current_period_start`. Bandingkan `used` vs `monthly_token_limit`.
  - ≥80% → sisipkan warning di respons.
  - ≥100% → tolak (hard block), arahkan upgrade. Jangan kirim ke model.
- **Setelah respons:** Claude balikin `usage.input_tokens` + `usage.output_tokens` sebenarnya. Insert 1 baris `token_usage` (bukan estimasi).
- Periode reset natural: query pakai `created_at >= current_period_start`, jadi ganti periode = geser tanggal, tak perlu reset kolom.

> `count_tokens` untuk gating pre-flight (cegah overshoot). Angka final dari `usage` di response — itu yang akurat, dan itu yang di-SUM.

## 5. Auth di Frontend (access token in-memory)

Access token disimpan di memori (bukan localStorage — anti-XSS, lihat 10-SECURITY #3). Konsekuensi: refresh halaman = token hilang. Alur:

```mermaid
sequenceDiagram
    actor User
    participant FE as Next.js (client)
    participant API as Go API

    Note over FE: App load / refresh halaman
    FE->>API: POST /auth/refresh (cookie HttpOnly refresh)
    alt refresh valid
        API-->>FE: access token baru (+ rotate refresh cookie)
        Note over FE: simpan access di memori, lanjut
    else invalid/expired
        API-->>FE: 401
        Note over FE: redirect ke /login
    end
```

- **Silent refresh saat app mount** (client component / provider) — sebelum render halaman terproteksi.
- **Refresh proaktif** sebelum access token 15m kedaluwarsa (timer), atau reaktif saat dapat 401 → refresh → retry sekali.

## 6. Konsumsi SSE di Frontend (gotcha)

`EventSource` browser **tidak bisa** POST atau kirim `Authorization` header. Endpoint chat = `POST` + Bearer + body. Maka FE **tidak pakai `EventSource`** — pakai `fetch` + baca `response.body` sebagai `ReadableStream`, parse baris `data:` manual. Ini wajib dicatat biar agent FE tidak mentok di `EventSource`.
