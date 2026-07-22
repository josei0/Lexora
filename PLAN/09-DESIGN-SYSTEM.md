# 09 — Design System (dari FE client)

Sumber kebenaran: `frontend/app/globals.css`. **Semua UI (landing + app) wajib pakai token ini.** Jangan pakai warna/ukuran mentah (`text-blue-600`, `#hex`) — pakai token semantik shadcn (`bg-primary`, `text-accent`, dst).

## 1. Identitas

- **Nama produk:** Lexora
- **Kesan:** firma hukum modern, premium, formal.
- Dibuat via v0.app. Stack: Next 16, React 19, Tailwind v4, shadcn (`style: base-nova`), `@base-ui/react`, lucide-react, **bun**.

## 2. Warna (oklch, light & dark sudah ada)

| Token | Light | Peran |
|-------|-------|-------|
| `background` | cream `oklch(0.98 0.006 85)` | latar utama |
| `foreground` | navy gelap `oklch(0.24 0.025 260)` | teks |
| `primary` | navy `oklch(0.29 0.045 258)` | aksi utama, header |
| `accent` | emas/gold `oklch(0.72 0.12 72)` | highlight, ring, bullet |
| `muted` / `muted-foreground` | abu hangat | teks sekunder |
| `card`, `popover` | cream terang | permukaan |
| `border`, `input` | `oklch(0.9 0.01 85)` | garis |
| `destructive` | merah | error |
| `chart-1..5` | gold→navy | grafik dashboard |
| `sidebar*` | set token sidebar | **layout dashboard** |

Dark mode: semua token sudah didefinisikan (`.dark` + `prefers-color-scheme`). Jangan bikin palet baru.

> **Aksen emas dipakai hemat** — highlight kecil (ring, bullet, eyebrow), bukan tombol besar. Navy = warna aksi dominan.

## 3. Tipografi

- **Body:** Inter (`--font-inter`, `font-sans`).
- **Heading:** Playfair Display (`--font-serif`) — dipakai untuk judul besar (`font-serif` di h1–h3).
- Eyebrow/label: `text-xs uppercase tracking-widest` warna `accent` atau `muted-foreground`.
- Sudah di-wire di `frontend/app/layout.tsx` via `next/font/google`.

## 4. Bentuk & spacing

- Radius kecil & tegas: `--radius: 0.25rem`. Kartu/tombol pakai `rounded-sm`.
- Section besar: `py-20 lg:py-28`, container `max-w-7xl px-6 lg:px-8`.
- Border tipis `border-border` buat memisah section.

## 5. Komponen

- Pakai **shadcn** (base-nova) + `@base-ui/react`. Tambah komponen via `bunx shadcn@latest add <x>`.
- Ikon: **lucide-react**.
- Alias: `@/components`, `@/components/ui`, `@/lib`, `@/hooks`.
- Yang sudah ada: `components/ui/button.tsx` + komponen landing (hero, capabilities, dll) + `/platform`.

## 6. Layout app (dashboard) — dibangun

Belum ada dari client, tapi token sidebar sudah disiapkan. Bangun pakai:
- **Sidebar** (nav: Dashboard, Chat, Knowledge Base, Workspace, Settings) → pakai token `sidebar*`.
- **Topbar** tipis (org switcher, user menu, indikator token usage).
- Konten pakai `card` + `chart-*` untuk dashboard statistik.
- Konsisten light/dark.

## 7. Aturan buat agent eksekutor

1. **Jangan rebuild landing.** Folder `frontend/` dipakai apa adanya, cukup lokalisasi konten (§8).
2. **Reuse token, jangan hardcode warna.** Kalau butuh warna baru, tambah sebagai token di `globals.css`, jangan tempel hex di komponen.
3. Heading besar → `font-serif`. Body → default.
4. Halaman app baru = folder route baru di `frontend/app/`, pakai komponen shadcn + token yang sama.
5. FE fungsional dulu; polish visual detail nanti (client bakal review).

## 8. Lokalisasi (template → Indonesia)

Konten FE sekarang **Inggris + konsep hukum US** (Shepardizing, jurisdiction, controlling authority) dan nama "Lexora". Yang harus diganti:
- `frontend/app/layout.tsx`: `lang="en"` → `lang="id"`, metadata title/description ke ID.
- Semua copy komponen landing & `/platform` → Indonesia + konteks hukum Indonesia (UU, PP, Putusan, PKPU, yurisprudensi, JDIHN/BPK sebagai sumber).
- 4 modul di `platform-modules.tsx` disesuaikan ke fitur nyata: Riset hukum, Analisis dokumen, Draft dokumen, Knowledge base privat.

> **Catatan i18n (update besar ke-2 = internasional):** landing ID dulu. Konten EN template sudah ada, jadi **bilingual landing = bonus murah** (client bilang "kalau bisa dua sekaligus makin bagus") — boleh bikin toggle ID/EN sederhana di landing kalau sempat, tapi bukan blocker. **App tetap ID** untuk MVP; **jangan** pasang framework i18n penuh dulu (YAGNI). Saat nulis komponen app baru, taruh teks yang gampang diekstrak nanti (hindari nyebar string di logika). Seam didokumentasikan, belum dibangun.
