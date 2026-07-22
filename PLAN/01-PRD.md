# 01 — PRD (Product Requirements Document)

**Produk: Lexora** — platform AI legal assistant.

## 1. Ringkasan

Platform AI berbasis web untuk profesional hukum: riset hukum, cari dasar hukum, analisis perkara, baca putusan, ringkasan, draft dokumen, cari yurisprudensi, kronologi perkara, persiapan sidang.

> Platform **tidak** memberi keputusan hukum. Membantu pengguna kerja lebih cepat, dengan jawaban yang selalu **menyertakan sumber (citation)**.

**Arah produk (2 update besar):**
1. **Sekarang — hukum Indonesia** (MVP & fokus utama).
2. **Nanti — hukum internasional** (masih "mungkin"). Tidak dibangun sekarang; titik ekstensi (kolom `jurisdiction`, teks UI mudah diekstrak) hanya didokumentasikan agar tidak terkunci.

## 2. Target User

Kantor hukum (firma). Model organization → banyak anggota. Multi-tenant, data terisolasi per organization.

## 3. Roles

| Role | Level | Kemampuan |
|------|-------|-----------|
| **Super Admin** | Platform | Kelola user, subscription, AI model, prompt, knowledge base, monitoring, dashboard statistik, billing. **Tidak** membaca isi chat/dokumen tenant (metadata & agregat saja) |
| **Org Admin** | Organization | Tambah anggota, kelola dokumen KB org, kelola workspace, lihat penggunaan anggota (metadata). **Tidak** membaca isi chat/dokumen pribadi anggota |
| **User** | Anggota | Chat AI, upload dokumen, analisis perkara, riwayat chat, download hasil. Chat & dokumen pribadinya privat |

> Matriks otoritas konkret (role × aksi × scope) + cara enforce ada di [10-SECURITY](10-SECURITY.md) §Matriks RBAC. Prinsip: **isi chat & dokumen pribadi privat penuh — admin pun tidak baca**, cuma metadata/statistik.

## 4. Fitur

### 4.1 Dashboard
Menampilkan: jumlah chat hari ini, penggunaan token, penggunaan AI, jumlah dokumen, total workspace, subscription, riwayat aktivitas.

### 4.2 Chat AI (mirip Claude/ChatGPT)
Chat, rename, delete, search, folder chat, pin chat, export PDF, export Word.

### 4.3 Knowledge Base
Admin upload PDF/DOCX/TXT (UU, Peraturan, Putusan, SOP, Peraturan Internal).
Diproses otomatis: `Upload → OCR → Chunking → Embedding → Qdrant`.

### 4.4 RAG
Saat user bertanya (mis. "Bagaimana syarat PKPU?"):
`User → Search Vector → Ambil dokumen → Kirim ke Claude → Jawaban + sumber`.
Jawaban wajib menyertakan referensi yang dipakai agar bisa diverifikasi.

### 4.5 Citation (wajib tiap jawaban bersumber)
Nama UU · nomor pasal · nomor putusan · halaman dokumen (jika ada) · sumber referensi.

**Kalau tidak ada dasar hukum relevan di KB** (search tidak nemu chunk di atas threshold, atau KB kosong): AI **tidak mengarang** dan tidak jawab dari pengetahuan umum. Balas jujur bahwa dasar hukum tidak ditemukan. Halusinasi hukum lebih bahaya daripada tidak menjawab. Detail perilaku di [06-FLOWS](06-FLOWS.md) §2.

## 5. Scope

### MVP (Rilis 1) — WAJIB
- **Landing page** (dari client, lokalisasi ke ID)
- **App shell**: login + layout dashboard (sidebar) pakai design system client
- Auth (email+password) + multi-tenant (organization, membership, role)
- Knowledge Base upload + pipeline ingestion (Go background worker)
- Chat AI + RAG + Citation (internal KB)
- Dashboard basic (statistik)
- Subscription **per organization (seat-based)**: org beli N seat, ditagih ke org (bukan per user). Tier **Demo (gratis) / Pro ($17/seat, Claude Sonnet)**, model beda per tier. Jatah token per seat; pemakaian di-track per user per pesan. Tambah anggota = pakai 1 seat
- **Manajemen user (MVP tanpa email):** org admin buatkan akun anggota + password sementara (anggota ganti saat login pertama). Reset password = via org admin / super admin. Undangan email + self-service reset = fase 2 (butuh email service)
- Export PDF/Word
- Audit log

### Fase 2 — TUNDA
- Flow hukum terdedikasi (analisis perkara, kronologi, persiapan sidang) — MVP cukup via chat + template prompt
- Template draft dokumen (gugatan, kontrak)
- Sumber data eksternal (JDIHN, BPK, Peraturan.go.id, Hukumonline) via n8n scheduled ingestion — detail di [11-EXTERNAL-SOURCES](11-EXTERNAL-SOURCES.md)
- Editor dokumen in-app
- Folder & pin chat lanjutan
- Tier Premium (Claude Opus) — MVP cukup Demo+Pro
- Payment gateway (MVP manual invoice)
- **Email service:** undangan anggota via email + self-service reset password + verifikasi email (butuh SMTP/provider, MVP pakai jalur admin)
- **Chat sharing antar-anggota** — MVP chat privat per user

## 6. Requirement Non-Fungsional

- **Keamanan:** data hukum sensitif → audit log, password hashing (argon2/bcrypt), JWT. Isolasi data per org di query layer. (Enkripsi at-rest & secret manager menyusul saat pindah ke server; localhost belum perlu.)
- **Deploy:** localhost (docker-compose) dulu. Cloud/Coolify setelah MVP.
- **Frontend:** design system dari client sudah ada (folder `frontend/`, lihat [09-DESIGN-SYSTEM](09-DESIGN-SYSTEM.md)). Landing tinggal dilokalisasi; halaman app dibangun pakai token yang sama.
- **Bahasa:** UI Indonesia (MVP). Internasional = update besar ke-2, belum dibangun.
- **Skala awal:** < beberapa ratus user, ribuan dokumen — single node Postgres + Qdrant cukup.
- **Logging:** structured JSON.

## 7. Batasan

- File upload: PDF/DOCX/TXT, maks 20MB/file.
- Limit token: soft warning di 80%, hard block di 100%.
- Chat: simpan sampai user hapus (soft delete → purge).
