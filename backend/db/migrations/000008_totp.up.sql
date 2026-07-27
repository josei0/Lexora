-- TOTP super_admin (update5 fase 2; spec update2 §4.9/4.11)
alter table users add column totp_secret text;
alter table users add column totp_confirmed_at timestamptz;
alter table users add column totp_last_step bigint not null default 0;

create table admin_recovery_codes (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    code_hash text not null,
    used_at timestamptz
);

create index idx_recovery_unused on admin_recovery_codes (user_id) where used_at is null;
