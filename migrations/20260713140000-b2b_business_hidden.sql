-- +migrate Up

-- Lets a tenant "delete" a business from their own list/map without touching
-- the shared scraped lead pool.
ALTER TABLE b2b_business_crm ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- +migrate Down
ALTER TABLE b2b_business_crm DROP COLUMN IF EXISTS hidden;
