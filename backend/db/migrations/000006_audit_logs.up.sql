CREATE TABLE audit_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        REFERENCES organizations(id) ON DELETE SET NULL,
    user_id    UUID        REFERENCES users(id) ON DELETE SET NULL,
    action     TEXT        NOT NULL,
    resource_id UUID,
    ip         TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_logs_org_created  ON audit_logs(org_id, created_at DESC);
CREATE INDEX audit_logs_created      ON audit_logs(created_at DESC);
