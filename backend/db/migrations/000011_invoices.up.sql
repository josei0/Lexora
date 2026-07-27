-- invoice per periode (update4 fase 2). Xendit (provider/provider_id/checkout_url)
-- menyusul fase 12; kolomnya disiapkan sekarang, NULL selama jalur manual.
create table invoices (
    id              uuid primary key default gen_random_uuid(),
    organization_id uuid not null references organizations(id) on delete cascade,
    plan_id         uuid not null references plans(id),
    type            text not null default 'subscription',  -- subscription | topup
    package_code    text,                                  -- topup paket kecil/besar
    seats           int not null check (seats >= 1),
    amount_idr      bigint not null,                    -- plan.price_idr * seats, dihitung server
    period_start    timestamptz not null,
    period_end      timestamptz not null,               -- period_end lama + 1 bulan saat paid
    status          text not null default 'pending',    -- pending | paid | expired | void
    provider        text,                               -- xendit (fase 12)
    provider_id     text,                               -- external id gateway
    checkout_url    text,
    paid_at         timestamptz,
    created_at      timestamptz not null default now()
);

-- riwayat per org
create index idx_invoices_org_created on invoices (organization_id, created_at desc);

-- satu invoice subscription pending per org: cegah tagihan renewal dobel dari
-- ticker + klik user barengan. Top-up (type='topup') tidak kena dedup ini.
create unique index uniq_invoices_org_pending on invoices (organization_id) where status = 'pending' and type = 'subscription';
