-- +migrate Up

-- Sales advisors (asesores comerciales).
CREATE TABLE IF NOT EXISTS b2b_advisors (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT,
    phone      TEXT,
    city       TEXT,
    active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Territorial zones (a zone belongs to a city and may be owned by an advisor).
CREATE TABLE IF NOT EXISTS b2b_zones (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    city       TEXT NOT NULL,
    advisor_id BIGINT REFERENCES b2b_advisors(id) ON DELETE SET NULL,
    color      TEXT NOT NULL DEFAULT '#2563eb',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_b2b_zones_city ON b2b_zones(city);
CREATE INDEX IF NOT EXISTS idx_b2b_zones_advisor ON b2b_zones(advisor_id);

-- CRM overlay for scraped businesses. The primary key is the business key,
-- derived from the scraped entry (place_id, falling back to cid).
CREATE TABLE IF NOT EXISTS b2b_business_crm (
    place_id   TEXT PRIMARY KEY,
    status     TEXT NOT NULL DEFAULT 'prospect',
    advisor_id BIGINT REFERENCES b2b_advisors(id) ON DELETE SET NULL,
    zone_id    BIGINT REFERENCES b2b_zones(id) ON DELETE SET NULL,
    title      TEXT,
    notes      TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_b2b_business_crm_status ON b2b_business_crm(status);
CREATE INDEX IF NOT EXISTS idx_b2b_business_crm_advisor ON b2b_business_crm(advisor_id);
CREATE INDEX IF NOT EXISTS idx_b2b_business_crm_zone ON b2b_business_crm(zone_id);

-- +migrate Down

DROP TABLE IF EXISTS b2b_business_crm;
DROP TABLE IF EXISTS b2b_zones;
DROP TABLE IF EXISTS b2b_advisors;
