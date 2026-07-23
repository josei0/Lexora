-- refresh tokens for jwt rotation
create table refresh_tokens (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references users(id) on delete cascade,
    token_hash text not null unique,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    created_at timestamptz not null default now()
);

create index idx_refresh_tokens_hash on refresh_tokens (token_hash);
create index idx_refresh_tokens_user on refresh_tokens (user_id);
