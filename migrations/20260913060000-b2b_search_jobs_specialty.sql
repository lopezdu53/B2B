-- +migrate Up

ALTER TABLE b2b_business_crm ADD COLUMN IF NOT EXISTS specialty TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS b2b_search_jobs (
    job_id     BIGINT PRIMARY KEY,
    tenant_id  BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rubro      TEXT NOT NULL DEFAULT '',
    specialty  TEXT NOT NULL DEFAULT '',
    ingested   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS b2b_search_jobs_tenant_idx ON b2b_search_jobs (tenant_id);

-- +migrate Down

DROP TABLE IF EXISTS b2b_search_jobs;
ALTER TABLE b2b_business_crm DROP COLUMN IF EXISTS specialty;
