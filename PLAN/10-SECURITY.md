# 10 — Security

Data hukum = sensitif. Keamanan basic dipasang **dari hari pertama** (logika app, bukan urusan deploy). Localhost sekarang, tapi app-level harus bener biar pindah server tinggal tambah TLS/WAF.

Prinsip: **jangan bikin kripto sendiri**, pakai library matang, default aman. Ponytail nggak berlaku buat security — ini bagian yang nggak boleh disederhanakan.

## Peta ancaman → mitigasi

| # | Ancaman | Mitigasi | Di mana | Status |
|---|---------|----------|---------|--------|
| 1 | **SQLi** | Parameterized query (pgx), zero string-concat SQL. Migration eksplisit | `repository/postgres` | MVP |
| 2 | **XSS** | React auto-escape; **sanitize output LLM** sebelum render markdown; no `dangerouslySetInnerHTML`; CSP | frontend + header | MVP |
| 3 | **CSRF** | Access token di Authorization header (bukan cookie); refresh token cookie `HttpOnly` + `SameSite=Strict` | `middleware/auth`, frontend | MVP |
| 4 | **DDoS** | Rate limit per-IP/user; body size limit; timeout. Prod: WAF/CDN di depan | `middleware/ratelimit` | app MVP, infra later |
| 5 | **HTTP Security Headers** | CSP, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy`, `Permissions-Policy`, HSTS (prod) | `middleware`, `next.config` | MVP (HSTS later) |
| 6 | **CORS** | Allowlist origin (`localhost:3000`, `app.localhost`, `admin.localhost`), bukan `*`. Credentials terkontrol | `middleware/cors` | MVP |
| 7 | **JWT Auth** | Access 15m, refresh rotation, verify `exp/iss/aud`, klaim minimal (`user_id/org_id/role`), secret kuat | `pkg/jwt`, `middleware/auth` | MVP |
| 8 | **IDOR** | **Selalu scope query ke `organization_id` + kepemilikan.** Jangan fetch by id doang | usecase + repo | MVP (kritis) |
| 9 | **SSRF** | Tanpa fetch URL user-supplied di MVP. Saat ingestion eksternal (fase 2): allowlist domain, blok IP internal | usecase, n8n | MVP-safe, fase 2 |
| 10 | **Credential Stuffing** | Rate-limit login, backoff progresif, error generik, argon2id, cek password bocor (opsional) | `usecase/auth`, `middleware` | MVP |
| 11 | **Dependency Vuln** | `govulncheck` (Go), `bun audit` (FE), update rutin | dev workflow | MVP |
| 12 | **Encryption at Rest** | Password argon2id; secret di `.env` (gitignore); disk encryption di server. Jangan roll crypto sendiri | seluruh app, deploy | MVP (disk enc. di server) |
| 13 | **Mass Assignment** | Selalu pakai DTO terpisah untuk bind request — jangan bind JSON user langsung ke struct DB/domain. Field sensitif (`role`, `is_active`, `plan_id`) tidak boleh ada di DTO user-facing | `delivery/dto`, handler | MVP |
| 14 | **Information Exposure** | Error handler global: log detail ke structured log, kirim ke client hanya pesan generik (`{"error":"internal server error"}`). Jangan expose stack trace, nama tabel, atau versi DB | `middleware/recover`, handler | MVP |
| 15 | **Business Logic Flaws** | Validasi nilai bisnis di usecase: token limit tidak boleh negatif, jumlah invoice tidak boleh ≤0, status transisi harus eksplisit. Negative test wajib untuk alur billing & subscription | `usecase/billing`, `usecase/subscription` | MVP |
| 16 | **Webhook Spoofing** | Setiap webhook masuk (dari n8n atau payment gateway nanti) wajib diverifikasi via HMAC signature di header. Shared secret dari `.env`, bukan hardcode | `handler/webhook_handler` | MVP |
| 17 | **Insufficient Logging** | Log semua event kritis: login (sukses/gagal), upload dokumen, perubahan role, error 5xx. Pakai `slog` (stdlib Go). Jangan log data sensitif (password, token, isi dokumen) | `pkg/logger`, middleware | MVP |
| 18 | **Broken Access Control** | RBAC eksplisit per endpoint (matriks di bawah). Cek role **dan** kepemilikan, bukan cuma "sudah login". Super/org admin tidak baca isi chat/dokumen user | `middleware/rbac`, usecase | MVP (kritis) |

## Catatan per item (yang non-obvious)

**2 — XSS di output AI.** Jawaban LLM/RAG bisa mengandung markup. Render markdown pakai renderer aman (no raw HTML) atau sanitize (mis. DOMPurify) sebelum tampil. Ini titik XSS paling gampang kelewat di app chat.

**3 — CSRF.** Karena API pakai Bearer token di header, request API kebal CSRF. Yang rawan cuma endpoint refresh (pakai cookie). Solusi: refresh cookie `HttpOnly; Secure; SameSite=Strict`, endpoint refresh khusus. Access token disimpan di memori frontend, bukan localStorage.

**7 — JWT.** Refresh token **rotation**: tiap refresh, token lama invalid (simpan jti/hash di DB untuk revoke). Logout = hapus refresh. Jangan taruh data sensitif di payload (base64, bukan enkripsi).

**8 — IDOR = ancaman #1 multi-tenant.** Contoh salah: `SELECT * FROM chats WHERE id = $1`. Benar: `... WHERE id = $1 AND organization_id = $2`. `organization_id` diambil dari JWT (`middleware/tenant`), **bukan** dari input user. Enforce di repo, uji dengan test lintas-org (wajib, lihat 08 §4).

**9 — SSRF.** MVP cuma upload file, nggak ada fetch URL dari user → risiko rendah. Saat fase 2 (ingestion JDIHN/BPK/Hukumonline via n8n): domain di-allowlist, dan tolak URL yang resolve ke IP privat/loopback/link-local (`127.0.0.0/8`, `10/8`, `172.16/12`, `192.168/16`, `169.254/16`). Webhook dari n8n ke backend divalidasi (shared secret).

**10 — Credential Stuffing.** Login error harus generik ("email atau password salah"), jangan bocorin field mana yang salah. Rate-limit per IP + per akun, backoff naik tiap gagal. CAPTCHA opsional kalau perlu.

**11 — Dependency.** Jalanin `govulncheck ./...` dan `bun audit` sebelum rilis. Nanti masuk CI. Jangan tambah dependency buat hal yang beberapa baris kode selesai (ponytail = permukaan serangan lebih kecil).

**12 — Encryption at Rest.** Localhost: cukup password ke-hash + `.env` nggak di-commit. Server: aktifkan disk/volume encryption (BitLocker/LUKS/cloud KMS). Kolom super-sensitif bisa dienkripsi app-level nanti, tapi dokumen hukum itu sendiri data-nya → andalkan disk encryption + kontrol akses (IDOR/JWT), bukan enkripsi kolom.

## Aturan keras (buat agent eksekutor)

1. **Query selalu parameterized** + **selalu scope `organization_id`**. Nggak ada pengecualian.
2. **Sanitize output AI** sebelum render.
3. **CORS allowlist**, jangan `*`. **Security headers** dipasang di middleware.
4. **JWT**: access pendek + refresh rotation. Password **argon2id**. Secret dari env.
5. **Rate-limit** login & endpoint mahal (chat/RAG).
6. Jangan log data sensitif (password, token, isi dokumen) ke structured log.
7. Cek `govulncheck` + `bun audit` sebelum nandain fase selesai.

> MVP fokus item app-level (1,2,3,5,6,7,8,10,11,13,14,15,16,17). Item infra-level (4 WAF, 12 disk encryption, HSTS) diaktifkan saat pindah server — tapi kait-nya (rate limit, header, hashing) udah dipasang sekarang.

## Default konkret (biar agent nggak nebak)

- **Rate limit:** login `5 req / menit / IP` + `10 req / menit / akun`; chat/RAG `20 req / menit / user`; upload `10 req / menit / user`. Store **in-memory** (cukup single-node localhost). `// ponytail: in-memory limiter, pindah Redis kalau multi-instance`.
- **Validasi upload:** cek MIME dengan **sniff isi file** (`http.DetectContentType` / magic bytes), bukan cuma ekstensi. Allowlist: PDF, DOCX, TXT. Tolak selain itu. Ukuran ≤20MB (body limit di middleware juga).
- **Body size limit global:** default `1MB` untuk JSON; endpoint upload khusus `20MB`.
- **Bentuk error response standar** (anti information exposure #14):
  ```json
  { "error": { "code": "invalid_request", "message": "pesan aman untuk user" } }
  ```
  Detail teknis (stack, query, driver) hanya ke structured log, tak pernah ke client. Map error domain → HTTP code di satu tempat (helper di delivery), jangan bocor error mentah.
- **Password sementara (MVP):** admin buat user → password acak sekali pakai, `must_change_password` → paksa ganti saat login pertama. Jangan kirim plain lewat kanal tak aman.

## Matriks RBAC (otoritas — acuan `middleware/rbac` + usecase)

3 role. **Super Admin** = flag `users.system_role`, lintas-platform, tak terikat org. **Org Admin** & **Member** = `memberships.role`, dalam satu org.

Prinsip privasi (diputuskan): **isi chat & dokumen pribadi user bersifat privat — tidak ada role lain yang bisa membacanya, termasuk admin.** Admin hanya lihat metadata/statistik (jumlah, token), bukan konten.

| Aksi | Super Admin | Org Admin | Member (User) |
|------|:-----------:|:---------:|:-------------:|
| **Platform** | | | |
| Kelola semua organization (CRUD) | ✅ | ❌ | ❌ |
| Kelola `plans`, `ai_models`, `prompts` | ✅ | ❌ | ❌ |
| Monitoring platform (agregat) | ✅ | ❌ | ❌ |
| **Organization** | | | |
| Tambah/nonaktif anggota (dalam org) | ❌¹ | ✅ | ❌ |
| Set role anggota (member↔org_admin) | ❌¹ | ✅ | ❌ |
| Reset password anggota | ✅ | ✅ (org-nya) | ❌ |
| Lihat penggunaan anggota (token, jumlah chat) | ✅ (metadata) | ✅ (metadata) | ❌ |
| Kelola subscription/seats org | ✅ | lihat saja | ❌ |
| **Knowledge Base (scope=knowledge_base)** | | | |
| Upload/hapus dokumen KB org | ❌² | ✅ | ❌ |
| Cari/pakai KB org saat chat | ❌² | ✅ | ✅ |
| **Dokumen pribadi (scope=user)** | | | |
| Upload/hapus dokumen sendiri | ❌² | ✅ (sendiri) | ✅ (sendiri) |
| Baca **isi** dokumen user lain | ❌ | ❌ | ❌ |
| **Chat** | | | |
| Buat/kirim/hapus chat sendiri | ❌² | ✅ (sendiri) | ✅ (sendiri) |
| Baca **isi** chat user lain | ❌ | ❌ | ❌ |
| Export chat sendiri (PDF/Word) | ❌² | ✅ | ✅ |

¹ Super Admin tidak jadi anggota org, jadi tidak kelola anggota via jalur org_admin. Manajemen user platform-level tetap bisa (nonaktif akun, reset password).
² Super Admin operasikan platform, **bukan** pemakai fitur hukum — tidak punya chat/KB/dokumen sendiri, dan tidak baca milik tenant.

**Cara enforce (jangan cuma di FE):**
1. `middleware/auth` → siapa (user_id, org_id, role dari JWT).
2. `middleware/rbac` → guard role kasar per route (mis. route `/admin/*` wajib `super_admin`).
3. **Usecase** → cek kepemilikan halus: `chat.user_id == caller.user_id`, `document.organization_id == caller.org_id`, dst. RBAC route saja **tidak cukup** — resource ownership dicek di usecase (ini yang cegah IDOR #8 + broken access #18).
4. Uji: test lintas-role (member akses route admin → 403) + lintas-user (baca chat orang lain → 404/403).
