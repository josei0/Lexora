-- documents + chunks for knowledge base ingestion
create table documents (
    id              uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    uploaded_by     uuid not null references users(id),
    scope           text not null default 'knowledge_base',  -- knowledge_base | user
    file_name       text not null,
    mime_type       text not null,
    file_size       bigint not null,
    storage_path    text not null,
    status          text not null default 'uploaded',  -- uploaded | processing | indexed | failed
    error           text,
    created_at      timestamptz not null default now(),
    updated_at      timestamptz not null default now()
);

create index idx_documents_org on documents (organization_id);
create index idx_documents_status on documents (status);

create table document_chunks (
    id              uuid primary key default gen_random_uuid(),
    document_id     uuid not null references documents(id) on delete cascade,
    chunk_index     int not null,
    page_no         int,
    qdrant_point_id text not null,
    created_at      timestamptz not null default now()
);

create index idx_document_chunks_doc on document_chunks (document_id);
