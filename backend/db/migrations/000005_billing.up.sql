-- plans, subscriptions (per org + seats), prompts
create table plans (
    id                  uuid primary key default gen_random_uuid(),
    code                text not null unique,           -- demo | pro
    name                text not null,
    model               text not null,                  -- maia model id
    price_idr           bigint not null default 0,      -- per seat / month
    monthly_token_limit bigint not null default 0,      -- per seat; 0 = unlimited
    is_active           boolean not null default true,
    created_at          timestamptz not null default now(),
    updated_at          timestamptz not null default now()
);

create table subscriptions (
    id              uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    plan_id         uuid not null references plans(id),
    seats           int not null default 1 check (seats >= 1),
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now(),
    unique (organization_id)                            -- satu subscription aktif per org
);

create table prompts (
    id         uuid primary key default gen_random_uuid(),
    key        text not null unique,                    -- 'system' dst
    content    text not null,
    updated_by uuid references users(id) on delete set null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
