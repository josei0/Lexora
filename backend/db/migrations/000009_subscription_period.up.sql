-- masa aktif langganan (update5 fase 6).
-- NULL = tanpa masa aktif (langganan lama + Demo) -> diperlakukan active selamanya,
-- jadi migrasi ini tidak mengunci org yang sudah jalan.
alter table subscriptions add column current_period_end timestamptz;
