drop index if exists idx_refresh_tokens_family;
alter table refresh_tokens drop column family_id;
