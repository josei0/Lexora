-- paket web search + cap harian (update5 fase 7 & 5)

-- asal-usul dokumen hasil ingest web; NULL = upload biasa
alter table documents add column source_url text;
create index idx_documents_source_url on documents (organization_id, source_url) where source_url is not null;

-- citation yang menunjuk web, bukan dokumen pustaka
alter table citations add column source_url text;

-- gating per plan
alter table plans add column web_search_enabled boolean not null default false;
alter table plans add column daily_web_searches int not null default 0; -- 0 = mati
alter table plans add column daily_messages int not null default 0;     -- 0 = tanpa cap

-- log pencarian: kuota harian + sumber data kandidat pustaka.
-- query = pertanyaan hukum user (sensitif) -> retensi 90 hari, lihat fase 8.
create table web_searches (
    id              uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    user_id         uuid not null references users(id) on delete cascade,
    query           text not null,
    provider        text not null,
    results_count   int not null default 0,
    top_urls        jsonb not null default '[]'::jsonb,
    created_at      timestamptz not null default now()
);

create index idx_web_searches_quota on web_searches (organization_id, user_id, created_at);
