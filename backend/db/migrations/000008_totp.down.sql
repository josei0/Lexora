drop table admin_recovery_codes;
alter table users drop column totp_secret;
alter table users drop column totp_confirmed_at;
alter table users drop column totp_last_step;
