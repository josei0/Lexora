create table chats (
    id uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    title text not null default 'Percakapan baru',
    is_pinned boolean not null default false,
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create index idx_chats_org_user on chats (organization_id, user_id, updated_at desc);

create table messages (
    id uuid primary key default gen_random_uuid(),
    chat_id uuid not null references chats(id) on delete cascade,
    role text not null check (role in ('user', 'assistant')),
    content text not null,
    model text,
    created_at timestamptz not null default now()
);
create index idx_messages_chat on messages (chat_id, created_at);

create table citations (
    id uuid primary key default gen_random_uuid(),
    message_id uuid not null references messages(id) on delete cascade,
    document_chunk_id uuid references document_chunks(id) on delete set null,
    document_id uuid,
    reference_label text not null,
    marker int not null default 0,
    page_no int,
    score real,
    created_at timestamptz not null default now()
);
create index idx_citations_message on citations (message_id);

create table token_usage (
    id uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    message_id uuid references messages(id) on delete set null,
    model text not null,
    input_tokens int not null default 0,
    output_tokens int not null default 0,
    created_at timestamptz not null default now()
);
create index idx_token_usage_user_created on token_usage (user_id, created_at);
