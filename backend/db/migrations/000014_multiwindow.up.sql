-- limit multi-window gaya Claude (update8): session 5-jam + weekly + monthly.
-- monthly_token_limit sudah ada (000005). 0 = window tak membatasi untuk plan itu.
alter table plans add column session_token_limit bigint not null default 0;
alter table plans add column weekly_token_limit  bigint not null default 0;

-- session window butuh state: kapan session 5-jam sekarang mulai (opsi B, update8 §2.2).
-- null = belum ada session aktif. Di-set saat pesan pertama / pasca-idle > 5 jam.
alter table subscriptions add column session_started_at timestamptz;
