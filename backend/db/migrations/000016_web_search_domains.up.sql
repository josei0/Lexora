-- allowlist domain web-search dari DB (update9-B). Sebelumnya env WEB_SEARCH_DOMAINS
-- statis saat boot; kini DB jadi sumber, env fallback seed.
create table web_search_domains (
    id         uuid primary key default gen_random_uuid(),
    host       text not null unique,
    created_at timestamptz not null default now()
);

-- seed 4 default gov (sama dgn default env). Di sini, bukan seed/main.go (hindari Agent A).
insert into web_search_domains (host) values
    ('peraturan.bpk.go.id'),
    ('peraturan.go.id'),
    ('jdihn.go.id'),
    ('putusan3.mahkamahagung.go.id')
on conflict (host) do nothing;
