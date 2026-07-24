-- family_id buat reuse detection.
-- default volatile: tiap baris lama dapat family unik.
alter table refresh_tokens add column family_id uuid not null default gen_random_uuid();

create index idx_refresh_tokens_family on refresh_tokens (user_id, family_id);
