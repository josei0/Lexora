-- dunning reminder (update9-A): anti-dobel-kirim per titik H-7/H-1/H+3.
-- null = belum pernah diingatkan. Di-set now tiap kirim reminder.
alter table subscriptions add column last_reminder_at timestamptz;
