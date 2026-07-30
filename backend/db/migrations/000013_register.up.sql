-- self-serve register + login Google (update6 fase 15-16).
-- verifikasi email = kolom di users (bukan tabel; resend tak diminta, u6 §5.2).
-- google_sub untuk login Google (u6 §5.3). Semua nullable -> aman utk row existing.
alter table users add column email_verified_at timestamptz;
alter table users add column verify_token_hash  text;
alter table users add column verify_expires_at  timestamptz;
alter table users add column google_sub         text unique;
