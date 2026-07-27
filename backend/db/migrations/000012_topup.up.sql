-- top-up kuota (u5 §3): satu baris per pembelian, bukan ledger saldo.
-- limit efektif = plan.monthly_token_limit * seats + SUM(topups window bulan ini).
create table quota_topups (
    id              uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    invoice_id      uuid not null unique references invoices(id), -- unique = idempoten mark-paid dobel
    tokens          bigint not null,
    created_at      timestamptz not null default now()
);

create index quota_topups_org_created on quota_topups(organization_id, created_at);
