-- init schema: organizations, users, memberships
create extension if not exists "pgcrypto";

create table organizations (
    id         uuid primary key default gen_random_uuid(),
    name       text not null,
    slug       text not null unique,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table users (
    id                   uuid primary key default gen_random_uuid(),
    email                text not null unique,
    password_hash        text not null,
    full_name            text not null,
    system_role          text not null default 'none',  -- super_admin | none
    is_active            boolean not null default true,
    must_change_password boolean not null default false,
    created_at           timestamptz not null default now(),
    updated_at           timestamptz not null default now()
);

create table memberships (
    id              uuid primary key default gen_random_uuid(),
    user_id         uuid not null references users(id) on delete cascade,
    organization_id uuid not null references organizations(id) on delete cascade,
    role            text not null default 'member',  -- org_admin | member
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now(),
    unique (user_id, organization_id)
);

create index idx_memberships_user_org on memberships (user_id, organization_id);
create index idx_memberships_org on memberships (organization_id);
