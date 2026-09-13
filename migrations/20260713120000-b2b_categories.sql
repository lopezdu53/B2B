-- +migrate Up

-- Per-tenant business categories (Restaurantes, Hoteles, Supermercados, …).
CREATE TABLE IF NOT EXISTS b2b_categories (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_b2b_categories_tenant ON b2b_categories(tenant_id);

-- A business (per tenant) can be filed under one category.
ALTER TABLE b2b_business_crm ADD COLUMN IF NOT EXISTS category_id BIGINT REFERENCES b2b_categories(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_b2b_business_crm_category ON b2b_business_crm(category_id);

-- Seed the default tenant with the common categories.
INSERT INTO b2b_categories (tenant_id, name) VALUES
    (1, 'Restaurantes'),
    (1, 'Hoteles'),
    (1, 'Supermercados')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- +migrate Down
ALTER TABLE b2b_business_crm DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS b2b_categories;
