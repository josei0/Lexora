# 11 — Sumber Hukum Eksternal (Fase 2)

Referensi buat ingestion eksternal (JDIHN/BPK/dll) — **belum dikerjakan di MVP.** MVP cuma pakai KB internal yang di-upload admin. Dokumen ini disimpan biar nggak hilang saat fase 2 digarap.

## Pusat data peraturan pemerintah

| Sumber | Keterangan |
|--------|------------|
| **JDIHN** (Jaringan Dokumentasi dan Informasi Hukum Nasional) | Dikelola Kemenkumham. Peraturan pusat, daerah, hingga putusan pengadilan |
| **Database Peraturan BPK** | Populer di praktisi buat cek UU dari awal sampai riwayat perubahan (amandemen) |
| **Peraturan.go.id** | Portal resmi pemerintah. Akses gratis UU, PP, Perpres |

## Portal analisis & praktisi

| Sumber | Keterangan |
|--------|------------|
| **Hukumonline** | Platform swasta, paling umum dipakai lawyer & akademisi. Peraturan lengkap + analisis, sejarah pasal, terjemahan |

## Cara ambil (rencana fase 2)

- **Lewat n8n scheduled**, bukan scraping live saat user nanya.
- Hasil masuk ke Knowledge Base (jadi dokumen KB scope org atau global), lewat pipeline ingestion yang sama: OCR → chunk → embed → Qdrant.
- **SSRF guard** (lihat [10-SECURITY](10-SECURITY.md) #9): allowlist domain di atas, tolak URL yang resolve ke IP internal.
- Cek legal/ToS tiap sumber sebelum otomatis narik (Hukumonline = swasta).
