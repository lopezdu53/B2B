-- +migrate Up

-- Tenants = the platform's client companies (each a retail B2B).
CREATE TABLE IF NOT EXISTS tenants (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Default tenant owns all pre-existing B2B data.
INSERT INTO tenants (id, name) VALUES (1, 'Default')
ON CONFLICT (id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('tenants', 'id'), GREATEST((SELECT MAX(id) FROM tenants), 1));

-- Users get a role and (for tenant users) a tenant + optional advisor link.
--   superadmin: platform owner, tenant_id NULL, sees/manages every tenant.
--   admin:      a client's administrator, scoped to their tenant.
--   advisor:    a client's sales rep, scoped to their tenant (future login).
ALTER TABLE users ADD COLUMN IF NOT EXISTS role       TEXT NOT NULL DEFAULT 'admin';
ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id  BIGINT REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS advisor_id BIGINT;

-- Existing users are the platform owners.
UPDATE users SET role = 'superadmin', tenant_id = NULL WHERE role = 'admin';

-- Scope B2B data to a tenant (existing rows belong to the default tenant).
ALTER TABLE b2b_advisors     ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE b2b_zones        ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE b2b_business_crm ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 1 REFERENCES tenants(id) ON DELETE CASCADE;

-- CRM overlay is per-tenant: the same scraped business can be a client for one
-- tenant and a prospect for another, so the key becomes (place_id, tenant_id).
ALTER TABLE b2b_business_crm DROP CONSTRAINT IF EXISTS b2b_business_crm_pkey;
ALTER TABLE b2b_business_crm ADD PRIMARY KEY (place_id, tenant_id);

CREATE INDEX IF NOT EXISTS idx_b2b_advisors_tenant     ON b2b_advisors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_b2b_zones_tenant        ON b2b_zones(tenant_id);
CREATE INDEX IF NOT EXISTS idx_b2b_business_crm_tenant ON b2b_business_crm(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant            ON users(tenant_id);

-- +migrate Down
ALTER TABLE b2b_business_crm DROP CONSTRAINT IF EXISTS b2b_business_crm_pkey;
ALTER TABLE b2b_business_crm ADD PRIMARY KEY (place_id);
ALTER TABLE b2b_business_crm DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE b2b_zones DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE b2b_advisors DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE users DROP COLUMN IF EXISTS advisor_id;
ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE users DROP COLUMN IF EXISTS role;
DROP TABLE IF EXISTS tenants;
